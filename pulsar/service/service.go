package service

import (
	"errors"
	"time"

	"sync"

	"github.com/dedis/cothority/pulsar/protocol"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/network"
)

// ServiceName denotes the name of the service.
const ServiceName = "RandHound"

var randhoundService onet.ServiceID

func init() {
	randhoundService, _ = onet.RegisterNewService(ServiceName, newService)
	network.RegisterMessage(propagateSetup{})
}

// Service is the main struct of the Pulsar service.
type Service struct {
	*onet.ServiceProcessor
	setup      bool
	nodes      int
	groups     int
	purpose    string
	randReady  chan bool
	randLock   sync.Mutex
	random     []byte
	transcript *protocol.Transcript
	interval   int
	tree       *onet.Tree
}

// Setup runs, upon request, the instantiation of the service.
func (s *Service) Setup(msg *SetupRequest) (*SetupReply, error) {

	// Service has already been setup, ignoring further setup requests
	if s.setup == true {
		return nil, errors.New("service already setup")
	}
	s.setup = true

	if msg.Interval <= 0 {
		return nil, errors.New("bad interval parameter")
	}

	s.tree = msg.Roster.GenerateBinaryTree()

	s.nodes = len(msg.Roster.List)
	s.groups = msg.Groups
	s.purpose = msg.Purpose
	s.interval = msg.Interval

	// This only locks the nodes but does not prevent from using them in
	// another RandHound setup.
	for _, n := range msg.Roster.List {
		if n.Public.Equal(s.Context.ServerIdentity().Public) {
			continue
		}
		if err := s.SendRaw(n, &propagateSetup{}); err != nil {
			return nil, err
		}
	}

	// Run RandHound in a loop
	go func() {
		for {
			s.run()
			time.Sleep(time.Duration(s.interval) * time.Millisecond)
		}
	}()
	<-s.randReady

	reply := &SetupReply{}
	return reply, nil
}

// Random accepts client randomness requests and returns the latest collective
// randomness together with the corresponding protocol transcript.
func (s *Service) Random(msg *RandRequest) (*RandReply, error) {

	s.randLock.Lock()
	defer s.randLock.Unlock()
	if s.setup == false || s.random == nil {
		return nil, errors.New("service not setup")
	}

	return &RandReply{
		R: s.random,
		T: s.transcript,
	}, nil
}

func (s *Service) propagate(env *network.Envelope) {
	s.setup = true
}

func (s *Service) run() {
	err := func() error {
		log.Lvl2("creating randomness")
		proto, err := s.CreateProtocol(ServiceName, s.tree)
		if err != nil {
			return err
		}
		rh := proto.(*protocol.RandHound)
		if err := rh.Setup(s.nodes, s.groups, s.purpose); err != nil {
			return err
		}

		if err := rh.Start(); err != nil {
			return err
		}

		select {
		case <-rh.Done:

			log.Lvlf1("done")

			random, transcript, err := rh.Random()
			if err != nil {
				return err
			}
			log.Lvlf1("collective randomness: ok")
			//log.Lvlf1("RandHound - collective randomness: %v", random)

			err = protocol.Verify(rh.Suite(), random, transcript)
			if err != nil {
				return err
			}
			log.Lvlf1("verification: ok")

			s.randLock.Lock()
			if s.random == nil {
				s.randReady <- true
			}
			s.random = random
			s.transcript = transcript
			s.randLock.Unlock()

		case <-time.After(time.Second * time.Duration(s.nodes) * 2):
			return err
		}
		return nil
	}()
	if err != nil {
		log.Error("while creating randomness:", err)
	}
}

type propagateSetup struct {
}

func newService(c *onet.Context) (onet.Service, error) {
	s := &Service{
		ServiceProcessor: onet.NewServiceProcessor(c),
		randReady:        make(chan bool),
	}
	if err := s.RegisterHandlers(s.Setup, s.Random); err != nil {
		return nil, errors.New("couldn't register message processing functions")
	}
	s.RegisterProcessorFunc(network.MessageType(propagateSetup{}), s.propagate)
	return s, nil
}