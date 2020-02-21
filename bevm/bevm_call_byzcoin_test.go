package bevm

import (
	"math/big"
	"testing"

	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/onet/v3/log"

	"github.com/stretchr/testify/require"
)

func Test_CallByzcoinContract(t *testing.T) {
	log.LLvl1("CallByzcoin")

	// Create a new ledger and prepare for proper closing
	bct := newBCTest(t)
	defer bct.Close()

	// Spawn a new BEvm instance
	instanceID, err := NewBEvm(bct.cl, bct.signer, bct.gDarc)
	require.Nil(t, err)

	// Create a new BEvm client
	bevmClient, err := NewClient(bct.cl, bct.signer, instanceID)
	require.Nil(t, err)

	// Initialize an account
	a, err := NewEvmAccount(testPrivateKeys[0])
	require.Nil(t, err)

	// Credit the account
	err = bevmClient.CreditAccount(big.NewInt(5*WeiPerEther), a.Address)
	require.Nil(t, err)

	// Deploy a CallByzcoin contract
	testID := byzcoin.NewInstanceID([]byte("test"))
	callBcContract, err := NewEvmContract("CallByzcoin",
		getContractData(t, "CallByzcoin", "abi"),
		getContractData(t, "CallByzcoin", "bin"))
	require.Nil(t, err)
	callBcInstance, err := bevmClient.Deploy(txParams.GasLimit,
		txParams.GasPrice, 0, a, callBcContract, testID, "value")
	require.Nil(t, err)

	// Spawn a value
	err = bevmClient.Transaction(txParams.GasLimit, txParams.GasPrice, 0, a,
		callBcInstance, "spawnValue", uint8(42))
	require.Nil(t, err)
}
