package multisig_test

import (
	"encoding/json"
	"testing"

	"github.com/EpiK-Protocol/go-epik/chain/actors"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin"
	"github.com/EpiK-Protocol/go-epik/chain/actors/builtin/power"
	"github.com/EpiK-Protocol/go-epik/chain/types"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	builtin0 "github.com/filecoin-project/specs-actors/v2/actors/builtin"
	multisig0 "github.com/filecoin-project/specs-actors/v2/actors/builtin/multisig"
	"github.com/stretchr/testify/assert"
)

type TxMsg struct {
	V int64  `json:"v,omitempty"`
	T string `json:"t,omitempty"`
	S string `json:"s,omitempty"`
	M TxData `json:"m,omitempty"`
}

type TxData struct {
	To     string `json:"to,omitempty"`
	Value  string `json:"value,omitempty"`
	Method int64  `json:"method,omitempty"`
	Params []byte `json:"params,omitempty"`
}

func TestMsg(t *testing.T) {

	value, aerr := types.ParseEPK("2676667")
	assert.Nil(t, aerr)
	// propose Investor
	to, err := address.NewFromString("f3qes5asp3flxi6w7ilwbnmdhqwcjdqxljbmuwof37dxq4q4nifdx52kitlnxe7nqohtpwp6euxth53loxlcia")
	assert.Nil(t, err)
	enc, actErr := actors.SerializeParams(&multisig0.ProposeParams{
		To:     to,
		Value:  types.BigInt(value),
		Method: 0,
		Params: nil,
	})
	assert.Nil(t, actErr)

	// &types.Message{
	// 	To:     msig,
	// 	From:   m.from,
	// 	Value:  abi.NewTokenAmount(0),
	// 	Method: builtin0.MethodsMultisig.Propose,
	// 	Params: enc,
	// }

	params := TxData{
		To:     builtin.TeamIDAddress.String(),
		Value:  "0",
		Method: int64(builtin0.MethodsMultisig.Propose),
		Params: enc,
	}
	msg := &TxMsg{
		V: 1,
		T: "deal",
		S: "bls",
		M: params,
	}

	mbytes, aerr := json.Marshal(msg)
	assert.Nil(t, aerr)
	// fmt.Print(string(mbytes))
	assert.NotNil(t, aerr, string(mbytes))
}

func TestRatio(t *testing.T) {

	sp, err := actors.SerializeParams(&power.ChangeWdPoStRatioParams{Ratio: 1000})
	assert.Nil(t, err)

	// propose Investor
	to := power.Address
	enc, actErr := actors.SerializeParams(&multisig0.ProposeParams{
		To:     to,
		Value:  abi.NewTokenAmount(0),
		Method: power.Methods.ChangeWdPoStRatio,
		Params: sp,
	})
	assert.Nil(t, actErr)

	// &types.Message{
	// 	To:     msig,
	// 	From:   m.from,
	// 	Value:  abi.NewTokenAmount(0),
	// 	Method: builtin0.MethodsMultisig.Propose,
	// 	Params: enc,
	// }

	params := TxData{
		To:     "f083",
		Value:  "0",
		Method: int64(builtin0.MethodsMultisig.Propose),
		Params: enc,
	}
	msg := &TxMsg{
		V: 1,
		T: "deal",
		S: "bls",
		M: params,
	}

	mbytes, aerr := json.Marshal(msg)
	assert.Nil(t, aerr)
	// fmt.Print(string(mbytes))
	assert.NotNil(t, aerr, string(mbytes))
}
