package stakepool

import (
	"encoding/json"

	"0chain.net/core/logging"
	"go.uber.org/zap"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/dbs/event"
)

type StakePoolReward dbs.StakePoolReward

func NewStakePoolReward(pId string, pType spenum.Provider) *StakePoolReward {
	var spu StakePoolReward
	spu.ProviderId = pId
	spu.ProviderType = int(pType)
	spu.DelegateRewards = make(map[string]int64)
	return &spu
}

func (spu StakePoolReward) Emit(
	tag event.EventTag,
	balances cstate.StateContextI,
) error {
	data, err := json.Marshal(&spu)
	if err != nil {
		return err
	}
	logging.Logger.Info("piers StakePoolReward Emit",
		zap.Any("event", spu))
	balances.EmitEvent(
		event.TypeStats,
		tag,
		spu.ProviderId,
		string(data),
	)
	return nil
}
