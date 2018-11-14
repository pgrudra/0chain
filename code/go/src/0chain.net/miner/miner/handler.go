package main

import (
	"fmt"
	"net/http"
	"strconv"

	"0chain.net/chain"
	. "0chain.net/logging"
	"0chain.net/node"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const updateConfigURL = "/v1/miner/updateConfig"
const updateConfigAllURL = "/v1/miner/updateConfigAll"

/*SetupHandlers - setup config related handlers */
func SetupHandlers() {
	http.HandleFunc(updateConfigURL, UpdateConfig)
	http.HandleFunc(updateConfigAllURL, UpdateConfigAll)
}

func UpdateConfig(w http.ResponseWriter, r *http.Request) {
	updateConfig(w, r, updateConfigURL)
}

func UpdateConfigAll(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		Logger.Error("failed to parse update config form", zap.Any("error", err))
		return
	}
	miners := chain.GetServerChain().Miners.Nodes
	for _, miner := range miners {
		if node.Self.PublicKey != miner.PublicKey {
			go func(miner *node.Node) {
				resp, err := http.PostForm(miner.GetN2NURLBase()+updateConfigURL, r.Form)
				if err != nil {
					Logger.Error("failed to update other miner's config", zap.Any("miner", miner.GetKey()), zap.Any("response", resp), zap.Any("error", err))
					return
				}
				defer resp.Body.Close()
			}(miner)
		}
	}
	updateConfig(w, r, updateConfigAllURL)
}

/*SetConfig*/
func updateConfig(w http.ResponseWriter, r *http.Request, updateUrl string) {
	newGenTimeout, _ := strconv.Atoi(r.FormValue("generate_timeout"))
	if newGenTimeout > 0 {
		chain.GetServerChain().SetGenerationTimeout(newGenTimeout)
		viper.Set("server_chain.block.generation.timeout", newGenTimeout)
	}
	newGenTxnRate, _ := strconv.ParseInt(r.FormValue("generate_txn"), 10, 32)
	if newGenTxnRate > 0 {
		SetTxnGenRate(int32(newGenTxnRate))
		viper.Set("server_chain.block.generation.transactions", newGenTxnRate)
	}
	newTxnWaitTime, _ := strconv.Atoi(r.FormValue("txn_wait_time"))
	if newTxnWaitTime > 0 {
		chain.GetServerChain().SetRetryWaitTime(newTxnWaitTime)
		viper.Set("server_chain.block.generation.retry_wait_time", newTxnWaitTime)
	}
	w.Header().Set("Content-Type", "text/html;charset=UTF-8")
	fmt.Fprintf(w, "<form action='%s' method='post'>", updateUrl)
	fmt.Fprintf(w, "Generation Timeout (time till a miner makes a block with less than max blocksize): <input type='text' name='generate_timeout' value='%v'><br>", viper.Get("server_chain.block.generation.timeout"))
	fmt.Fprintf(w, "Transaction Generation Rate (rate the miner will add transactions to create a block): <input type='text' name='generate_txn' value='%v'><br>", viper.Get("server_chain.block.generation.transactions"))
	fmt.Fprintf(w, "Retry Wait Time (time miner waits if there aren't enough transactions to reach max blocksize): <input type='text' name='txn_wait_time' value='%v'><br>", viper.Get("server_chain.block.generation.retry_wait_time"))
	fmt.Fprintf(w, "<input type='submit' value='Submit'>")
	fmt.Fprintf(w, "</form>")
}
