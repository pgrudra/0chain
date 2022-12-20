package minersc

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/smartcontract/provider"
	"0chain.net/smartcontract/provider/spenum"
	"0chain.net/smartcontract/stakepool"
	"github.com/0chain/common/core/util"
)

func (msc *MinerSmartContract) addToDelegatePool(t *transaction.Transaction,
	input []byte, gn *GlobalNode, balances cstate.StateContextI) (
	resp string, err error) {

	return stakepool.StakePoolLock(t, input, balances, msc.getStakePoolAdapter)
}

// getStakePool of given blobber
func (msc *MinerSmartContract) getStakePoolAdapter(pType spenum.Provider, providerID string,
	balances cstate.CommonStateContextI) (sp provider.AbstractStakePool, err error) {
	var mn *MinerNode
	switch pType {
	case spenum.Miner:
		mn, err = getMinerNode(providerID, balances)
	case spenum.Sharder:
		mn, err = getSharderNode(providerID, balances)
	default:
		return mn, common.NewErrorf("get_stake_pool",
			"unknown provider type")
	}
	switch err {
	case nil:
	case util.ErrValueNotPresent:
		return mn, common.NewErrorf("get_stake_pool",
			"miner not found or genesis miner used")
	default:
		return mn, common.NewErrorf("get_stake_pool",
			"unexpected DB error: %v", err)
	}
	return mn, nil
}

func (msc *MinerSmartContract) deleteFromDelegatePool(
	t *transaction.Transaction, inputData []byte, gn *GlobalNode,
	balances cstate.StateContextI) (resp string, err error) {

	return stakepool.StakePoolUnlock(t, inputData, balances, msc.getStakePoolAdapter)
}
