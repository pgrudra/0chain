package miner

import (
	"context"

	. "0chain.net/logging"
	"0chain.net/round"
	"go.uber.org/zap"
)

/*HandleVRFShare - handles the vrf share */
func (mc *Chain) HandleVRFShare(ctx context.Context, msg *BlockMessage) {
	mr := mc.GetMinerRound(msg.VRFShare.Round)
	if mr == nil {
		pr := mc.GetMinerRound(msg.VRFShare.Round - 1)
		if pr != nil {
			//This can happen because other nodes are slightly ahead. It is ok.
			mr = mc.StartNextRound(ctx, pr)
		} else {
			Logger.Error("handle vrf share - no prior round", zap.Int64("round", msg.VRFShare.Round))
			// We can't really provide a VRF share as we don't know the previous round's random number but we can collect the shares
			var r = round.NewRound(msg.VRFShare.Round)
			mr = mc.CreateRound(r)
			mr = mc.AddRound(mr).(*Round)
		}
	}
	if mr != nil {
		mc.AddVRFShare(ctx, mr, msg.VRFShare)
	}
}

/*HandleVerifyBlockMessage - handles the verify block message */
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage) {
	b := msg.Block
	if b.Round < mc.CurrentRound-1 {
		Logger.Debug("verify block (round mismatch)", zap.Int64("current_round", mc.CurrentRound), zap.Int64("block_round", b.Round))
		return
	}
	mr := mc.GetMinerRound(b.Round)
	if mr == nil {
		Logger.Error("handle verify block - got block proposal before starting round", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("miner", b.MinerID))
		pr := mc.GetMinerRound(b.Round - 1)
		if pr != nil {
			//If this happens, need to check
			mr = mc.StartNextRound(ctx, pr)
		} else {
			var r = round.NewRound(b.Round)
			mr = mc.CreateRound(r)
			mr = mc.AddRound(mr).(*Round)
		}
		//TODO: Byzantine
		mc.setRandomSeed(ctx, mr, b.RoundRandomSeed)
	} else {
		if !mr.IsVRFComplete() {
			//TODO: Byzantine
			mc.setRandomSeed(ctx, mr, b.RoundRandomSeed)
		}
		vts := mr.GetVerificationTickets(b.Hash)
		if len(vts) > 0 {
			mc.MergeVerificationTickets(ctx, b, vts)
			if b.IsBlockNotarized() {
				b = mc.AddRoundBlock(mr, b)
				mc.checkBlockNotarization(ctx, mr, b)
				return
			}
		}
	}
	if mr != nil {
		if !mc.ValidGenerator(mr.Round, b) {
			Logger.Error("Not a valid generator. Ignoring block with hash = " + b.Hash)
			return
		}
		Logger.Info("Added block to Round with hash = " + b.Hash)
		mc.AddToRoundVerification(ctx, mr, b)
	} else {
		Logger.Error("this should not happen %v", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("cround", mc.CurrentRound))
	}
}

/*HandleVerificationTicketMessage - handles the verification ticket message */
func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage) {
	var err error
	mr := msg.Round
	if mr == nil {
		mr = mc.GetMinerRound(msg.BlockVerificationTicket.Round)
		if mr == nil {
			pr := mc.GetMinerRound(msg.BlockVerificationTicket.Round - 1)
			if pr != nil {
				//This means, this node is way behind other nodes.
				mr = mc.StartNextRound(ctx, pr)
			} else {
				var r = round.NewRound(msg.BlockVerificationTicket.Round)
				mr = mc.CreateRound(r)
				mr = mc.AddRound(mr).(*Round)
			}
		}
	}
	b, err := mc.GetBlock(ctx, msg.BlockVerificationTicket.BlockID)
	if err != nil {
		if mr != nil {
			err = mc.VerifyTicket(ctx, msg.BlockVerificationTicket.BlockID, &msg.BlockVerificationTicket.VerificationTicket)
			if err != nil {
				Logger.Debug("verification ticket", zap.Error(err))
				return
			}
			mr.AddVerificationTicket(msg.BlockVerificationTicket)
			return
		}
		return
	}
	if b.Round < mc.LatestFinalizedBlock.Round {
		Logger.Debug("verification message (round mismatch)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("finalized_round", mc.LatestFinalizedBlock.Round))
		return
	}
	err = mc.VerifyTicket(ctx, b.Hash, &msg.BlockVerificationTicket.VerificationTicket)
	if err != nil {
		Logger.Debug("verification ticket", zap.Error(err))
		return
	}
	mc.ProcessVerifiedTicket(ctx, mr, b, &msg.BlockVerificationTicket.VerificationTicket)
}

/*HandleNotarizationMessage - handles the block notarization message */
func (mc *Chain) HandleNotarizationMessage(ctx context.Context, msg *BlockMessage) {
	if msg.Notarization.Round < mc.LatestFinalizedBlock.Round {
		Logger.Debug("notarization message", zap.Int64("round", msg.Notarization.Round), zap.Int64("finalized_round", mc.LatestFinalizedBlock.Round), zap.String("block", msg.Notarization.BlockID))
		return
	}
	r := mc.GetMinerRound(msg.Notarization.Round)
	if r == nil {
		if msg.ShouldRetry() {
			Logger.Error("notarization receipt handler (round not started yet) retrying", zap.String("block", msg.Notarization.BlockID), zap.Int8("retry_count", msg.RetryCount))
			msg.Retry(mc.BlockMessageChannel)
		} else {
			Logger.Error("notarization receipt handler (round not started yet)", zap.String("block", msg.Notarization.BlockID), zap.Int8("retry_count", msg.RetryCount))
		}
		return
	}
	msg.Round = r
	b, err := mc.GetBlock(ctx, msg.Notarization.BlockID)
	if err != nil {
		mc.AsyncFetchNotarizedBlock(msg.Notarization.BlockID)
		return
	}
	vts := b.UnknownTickets(msg.Notarization.VerificationTickets)
	if len(vts) == 0 {
		return
	}
	go mc.MergeNotarization(ctx, r, b, vts)
}

/*HandleNotarizedBlockMessage - handles a notarized block for a previous round*/
func (mc *Chain) HandleNotarizedBlockMessage(ctx context.Context, msg *BlockMessage) {
	mb := msg.Block
	mr := mc.GetMinerRound(mb.Round)
	if mr == nil {
		Logger.Error("handle notarized block message", zap.Int64("round", mb.Round))
		var r = round.NewRound(mb.Round)
		//TODO: Byzantine
		mr = mc.CreateRound(r)
		mr = mc.AddRound(mr).(*Round)
		mc.setRandomSeed(ctx, mr, mb.RoundRandomSeed)
	} else {
		nb := mr.GetNotarizedBlocks()
		for _, blk := range nb {
			if blk.Hash == mb.Hash {
				return
			}
		}
		if !mr.IsVRFComplete() {
			//TODO: Byzantine
			mc.setRandomSeed(ctx, mr, mb.RoundRandomSeed)
		}
	}
	b := mc.AddRoundBlock(mr, mb)
	if !mc.AddNotarizedBlock(ctx, mr, b) {
		return
	}
	mc.StartNextRound(ctx, mr)
}
