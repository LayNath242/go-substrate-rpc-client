// Go Substrate RPC Client (GSRPC) provides APIs and types around Polkadot and any Substrate-based chain RPC calls
//
// Copyright 2019 Centrifuge GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package teste2e

import (
	"fmt"
	"testing"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client"
	"github.com/centrifuge/go-substrate-rpc-client/config"
	"github.com/centrifuge/go-substrate-rpc-client/types"
	"github.com/stretchr/testify/assert"
)

func TestEnd2end(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end test in short mode.")
	}

	api, err := gsrpc.NewSubstrateAPI(config.Default().RPCURL)
	assert.NoError(t, err)

	fmt.Println()
	fmt.Printf("Connected to node: %v\n", api.Client.URL())
	fmt.Println()

	runtimeVersion, err := api.RPC.State.GetRuntimeVersionLatest()
	assert.NoError(t, err)
	fmt.Printf("authoringVersion: %v\n", runtimeVersion.AuthoringVersion)
	fmt.Printf("specVersion: %v\n", runtimeVersion.SpecVersion)
	fmt.Printf("implVersion: %v\n", runtimeVersion.ImplVersion)
	fmt.Println()

	hash, err := api.RPC.Chain.GetBlockHashLatest()
	assert.NoError(t, err)
	fmt.Printf("Latest block: %v\n", hash.Hex())
	fmt.Printf("\tView in Polkadot/Substrate Apps: https://polkadot.js.org/apps/#/explorer/query/%v?"+
		"rpc=wss://serinus-5.kusama.network\n", hash.Hex())
	fmt.Printf("\tView in polkascan.io: https://polkascan.io/pre/kusama-cc2/block/%v\n", hash.Hex())
	fmt.Println()

	header, err := api.RPC.Chain.GetHeader(hash)
	assert.NoError(t, err)
	fmt.Printf("Block number: %v\n", header.Number)
	fmt.Printf("Parent hash: %v\n", header.ParentHash.Hex())
	fmt.Printf("State root: %v\n", header.StateRoot.Hex())
	fmt.Printf("Extrinsics root: %v\n", header.ExtrinsicsRoot.Hex())
	fmt.Println()

	block, err := api.RPC.Chain.GetBlock(hash)
	assert.NoError(t, err)
	fmt.Printf("Total extrinsics: %v\n", len(block.Block.Extrinsics))
	fmt.Println()

	finHead, err := api.RPC.Chain.GetFinalizedHead()
	assert.NoError(t, err)
	fmt.Printf("Last finalized block in the canon chain: %v\n", finHead.Hex())
	fmt.Println()

	meta, err := api.RPC.State.GetMetadataLatest()
	assert.NoError(t, err)

	key, err := types.CreateStorageKey(meta, "Session", "Validators", nil)
	assert.NoError(t, err)

	var validators []types.AccountID
	err = api.RPC.State.GetStorageLatest(key, &validators)
	assert.NoError(t, err)
	fmt.Printf("Current validators:\n")
	for i, v := range validators {
		fmt.Printf("\tValidator %v: %#x\n", i, v)
	}
	fmt.Println()
}

func TestState_SubscribeStorage_EventsRaw(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end test in short mode.")
	}

	api, err := gsrpc.NewSubstrateAPI(config.Default().RPCURL)
	assert.NoError(t, err)

	key := types.NewStorageKey(types.MustHexDecodeString("0xcc956bdb7605e3547539f321ac2bc95c"))

	sub, err := api.RPC.State.SubscribeStorageRaw([]types.StorageKey{key})
	assert.NoError(t, err)

	timeout := time.After(10 * time.Second)

	for {
		select {
		case set := <-sub.Chan():
			fmt.Printf("%#v\n", set)
		case <-timeout:
			return // TODO unsubscribe/cleanup
		}
	}
}

func TestState_SubscribeStorage_Events(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end test in short mode.")
	}

	api, err := gsrpc.NewSubstrateAPI(config.Default().RPCURL)
	assert.NoError(t, err)

	meta, err := api.RPC.State.GetMetadataLatest()
	assert.NoError(t, err)

	key, err := types.CreateStorageKey(meta, "System", "Events", nil)
	assert.NoError(t, err)

	sub, err := api.RPC.State.SubscribeStorageRaw([]types.StorageKey{key})
	assert.NoError(t, err)

	timeout := time.After(10 * time.Second)

	for {
		select {
		case set := <-sub.Chan():
			fmt.Printf("%#v\n", set)
			for _, chng := range set.Changes {
				if !types.Eq(chng.StorageKey, key) || !chng.HasStorageData {
					// skip, we are only interested in events with content
					continue
				}
				events := types.EventRecords{}
				err = types.EventRecordsRaw(chng.StorageData).DecodeEventRecords(meta, &events)
				assert.NoError(t, err)

				fmt.Printf("%#v\n", events)
			}
		case <-timeout:
			return // TODO unsubscribe/cleanup
		}
	}
}