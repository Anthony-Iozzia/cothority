package bevm

import (
	"math/big"
	"testing"
	"time"

	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"golang.org/x/xerrors"

	"github.com/stretchr/testify/require"
)

func init() {
	err := byzcoin.RegisterGlobalContract(valContractID,
		valContractFromBytes)
	if err != nil {
		log.ErrFatal(err)
	}
}

const valContractID = "TestValContract"

func valContractFromBytes(in []byte) (byzcoin.Contract, error) {
	return valContract{value: in}, nil
}

// The test value contracts just holds a value
type valContract struct {
	byzcoin.BasicContract
	value []byte
}

func (c valContract) Spawn(rst byzcoin.ReadOnlyStateTrie,
	inst byzcoin.Instruction, coins []byzcoin.Coin) (sc []byzcoin.StateChange,
	cout []byzcoin.Coin, err error) {
	cout = coins

	// Find the darcID for this instance.
	var darcID darc.ID
	_, _, _, darcID, err = rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return
	}

	sc = []byzcoin.StateChange{
		byzcoin.NewStateChange(byzcoin.Create, inst.DeriveID(""),
			valContractID, inst.Spawn.Args.Search("value"), darcID),
	}
	return
}

func (c valContract) Invoke(rst byzcoin.ReadOnlyStateTrie,
	inst byzcoin.Instruction, coins []byzcoin.Coin) (sc []byzcoin.StateChange,
	cout []byzcoin.Coin, err error) {
	cout = coins

	// Find the darcID for this instance.
	var darcID darc.ID

	_, _, _, darcID, err = rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return
	}

	switch inst.Invoke.Command {
	case "update":
		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
				valContractID, inst.Invoke.Args.Search("value"), darcID),
		}
		return
	default:
		return nil, nil, xerrors.New("Value contract can only update")
	}
}
func Test_CallByzcoinContract(t *testing.T) {
	log.LLvl1("CallByzcoin")

	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	signer := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(3, true)

	// Initialize DARC with rights for BEvm and spawning a value contract
	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{
			"spawn:" + ContractBEvmID,
			"invoke:" + ContractBEvmID + ".credit",
			"invoke:" + ContractBEvmID + ".transaction",
			"spawn:" + valContractID,
		}, signer.Identity(),
	)
	require.Nil(t, err)

	gDarc := &genesisMsg.GenesisDarc
	genesisMsg.BlockInterval = time.Second

	// Create new ledger
	cl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.Nil(t, err)

	// // Create a new ledger and prepare for proper closing
	// bct := newBCTest(t)
	// defer bct.Close()

	// Spawn a new BEvm instance
	instanceID, err := NewBEvm(cl, signer, gDarc)
	require.Nil(t, err)

	// Create a new BEvm client
	bevmClient, err := NewClient(cl, signer, instanceID)
	require.Nil(t, err)

	// Initialize an account
	a, err := NewEvmAccount(testPrivateKeys[0])
	require.Nil(t, err)

	// Credit the account
	err = bevmClient.CreditAccount(big.NewInt(5*WeiPerEther), a.Address)
	require.Nil(t, err)

	// Deploy a CallByzcoin contract
	darcID := byzcoin.NewInstanceID(gDarc.GetBaseID())
	callBcContract, err := NewEvmContract("CallByzcoin",
		getContractData(t, "CallByzcoin", "abi"),
		getContractData(t, "CallByzcoin", "bin"))
	require.Nil(t, err)
	callBcInstance, err := bevmClient.Deploy(txParams.GasLimit,
		txParams.GasPrice, 0, a, callBcContract, darcID, valContractID)
	require.Nil(t, err)

	// Spawn a value
	err = bevmClient.Transaction(txParams.GasLimit, txParams.GasPrice, 0, a,
		callBcInstance, "spawnValue", uint8(42))
	require.Nil(t, err)
}
