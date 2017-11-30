package skipchain_test

import (
	"testing"

	"crypto/sha512"

	"bytes"

	"github.com/dedis/cothority/bftcosi"
	"github.com/dedis/cothority/skipchain"
	"github.com/dedis/kyber/util/random"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkipBlock_GetResponsible(t *testing.T) {
	l := onet.NewTCPTest(tSuite)
	_, roster, _ := l.GenTree(3, true)
	defer l.CloseAll()
	sbm := skipchain.NewSkipBlockMap()
	root0 := skipchain.NewSkipBlock()
	root0.Roster = roster
	root0.Hash = root0.CalculateHash()
	root0.BackLinkIDs = []skipchain.SkipBlockID{root0.Hash}
	sbm.Store(root0)
	root1 := root0.Copy()
	root1.Index++
	sbm.Store(root1)
	inter0 := skipchain.NewSkipBlock()
	inter0.ParentBlockID = root1.Hash
	inter0.Roster = roster
	inter0.Hash = inter0.CalculateHash()
	sbm.Store(inter0)
	inter1 := inter0.Copy()
	inter1.Index++
	inter1.BackLinkIDs = []skipchain.SkipBlockID{inter0.Hash}

	b, err := sbm.GetResponsible(root0)
	log.ErrFatal(err)
	assert.True(t, root0.Equal(b))

	b, err = sbm.GetResponsible(root1)
	log.ErrFatal(err)
	assert.True(t, root0.Equal(b))

	b, err = sbm.GetResponsible(inter0)
	log.ErrFatal(err)
	assert.Equal(t, root1.Hash, b.Hash)

	b, err = sbm.GetResponsible(inter1)
	log.ErrFatal(err)
	assert.True(t, inter0.Equal(b))
}

func TestSkipBlock_VerifySignatures(t *testing.T) {
	l := onet.NewTCPTest(tSuite)
	_, roster3, _ := l.GenTree(3, true)
	defer l.CloseAll()
	roster2 := onet.NewRoster(roster3.List[0:2])
	sbm := skipchain.NewSkipBlockMap()
	root := skipchain.NewSkipBlock()
	root.Roster = roster2
	root.BackLinkIDs = append(root.BackLinkIDs, skipchain.SkipBlockID{1, 2, 3, 4})
	root.Hash = root.CalculateHash()
	sbm.Store(root)
	log.ErrFatal(root.VerifyForwardSignatures())
	log.ErrFatal(sbm.VerifyLinks(root))

	block1 := root.Copy()
	block1.BackLinkIDs = append(block1.BackLinkIDs, root.Hash)
	block1.Index++
	sbm.Store(block1)
	require.Nil(t, block1.VerifyForwardSignatures())
	require.NotNil(t, sbm.VerifyLinks(block1))
}

func TestSkipBlock_Hash1(t *testing.T) {
	sbd1 := skipchain.NewSkipBlock()
	sbd1.Data = []byte("1")
	sbd1.Height = 4
	h1 := sbd1.UpdateHash()
	assert.Equal(t, h1, sbd1.Hash)

	sbd2 := skipchain.NewSkipBlock()
	sbd2.Data = []byte("2")
	sbd1.Height = 2
	h2 := sbd2.UpdateHash()
	assert.NotEqual(t, h1, h2)
}

func TestSkipBlock_Hash2(t *testing.T) {
	local := onet.NewLocalTest(tSuite)
	hosts, el, _ := local.GenTree(2, false)
	defer local.CloseAll()
	sbd1 := skipchain.NewSkipBlock()
	sbd1.Roster = el
	sbd1.Height = 1
	h1 := sbd1.UpdateHash()
	assert.Equal(t, h1, sbd1.Hash)

	sbd2 := skipchain.NewSkipBlock()
	sbd2.Roster = local.GenRosterFromHost(hosts[0])
	sbd2.Height = 1
	h2 := sbd2.UpdateHash()
	assert.NotEqual(t, h1, h2)
}

func TestBlockLink_Copy(t *testing.T) {
	// Test if copy is deep or only shallow
	b1 := &skipchain.BlockLink{}
	b1.Signature = []byte{1}
	b2 := b1.Copy()
	b2.Signature = []byte{2}
	if bytes.Equal(b1.Signature, b2.Signature) {
		t.Fatal("They should not be equal")
	}

	sb1 := skipchain.NewSkipBlock()
	sb1.ChildSL = append(sb1.ChildSL, []byte{3})
	sb2 := sb1.Copy()
	sb1.ChildSL[0] = []byte{1}
	sb2.ChildSL[0] = []byte{2}
	if bytes.Equal(sb1.ChildSL[0], sb2.ChildSL[0]) {
		t.Fatal("They should not be equal")
	}
	sb1.Height = 10
	sb2.Height = 20
	if sb1.Height == sb2.Height {
		t.Fatal("Should not be equal")
	}
}

func TestSign(t *testing.T) {
	l := onet.NewTCPTest(tSuite)
	servers, roster, _ := l.GenTree(10, true)
	msg := sha512.New().Sum(nil)
	sig, err := sign(msg, servers, l)
	log.ErrFatal(err)
	log.ErrFatal(sig.Verify(tSuite, roster.Publics()))
	sig.Msg = sha512.New().Sum([]byte{1})
	require.NotNil(t, sig.Verify(tSuite, roster.Publics()))
	defer l.CloseAll()
}

func sign(msg skipchain.SkipBlockID, servers []*onet.Server, l *onet.LocalTest) (*bftcosi.BFTSignature, error) {
	aggScalar := tSuite.Scalar().Zero()
	aggPoint := tSuite.Point().Null()
	for _, s := range servers {
		aggScalar.Add(aggScalar, l.GetPrivate(s))
		aggPoint.Add(aggPoint, s.ServerIdentity.Public)
	}
	rand := tSuite.Scalar().Pick(random.Stream)
	comm := tSuite.Point().Mul(rand, nil)
	sigC, err := comm.MarshalBinary()
	if err != nil {
		return nil, err
	}
	hash := sha512.New()
	hash.Write(sigC)
	aggPoint.MarshalTo(hash)
	hash.Write(msg)
	challBuff := hash.Sum(nil)
	chall := tSuite.Scalar().SetBytes(challBuff)
	resp := tSuite.Scalar().Mul(aggScalar, chall)
	resp = resp.Add(rand, resp)
	sigR, err := resp.MarshalBinary()
	if err != nil {
		return nil, err
	}
	sig := make([]byte, 64+len(servers)/8)
	copy(sig[:], sigC)
	copy(sig[32:64], sigR)
	return &bftcosi.BFTSignature{Sig: sig, Msg: msg, Exceptions: nil}, nil
}
