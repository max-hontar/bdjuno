package main

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"

	"github.com/forbole/juno/v4/modules/messages"
)

// CustomBasicManager represents a BasicManager for the custom chain
func CustomBasicManager() module.BasicManager {
	return module.NewBasicManager(
		auth.AppModuleBasic{},
		// authzmodule.AppModuleBasic{},
		genutil.AppModuleBasic{},
		bank.AppModuleBasic{},
		// capability.AppModuleBasic{},
		staking.AppModuleBasic{},
		// mint.AppModuleBasic{},
		// distr.AppModuleBasic{},
		gov.NewAppModuleBasic(getGovProposalHandlers()),
		params.AppModuleBasic{},
		// crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		// feegrantmodule.AppModuleBasic{},
		// groupmodule.AppModuleBasic{},
		// ibc.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		// evidence.AppModuleBasic{},
		// transfer.AppModuleBasic{},
		// ica.AppModuleBasic{},
		vesting.AppModuleBasic{},
	)
}

func getGovProposalHandlers() []govclient.ProposalHandler {
	var govProposalHandlers []govclient.ProposalHandler

	govProposalHandlers = append(govProposalHandlers,
		paramsclient.ProposalHandler,
		// distrclient.ProposalHandler,
		upgradeclient.LegacyProposalHandler,
		upgradeclient.LegacyCancelProposalHandler,
		// ibcclientclient.UpdateClientProposalHandler,
		// ibcclientclient.UpgradeProposalHandler,
	)

	return govProposalHandlers
}

// CustomAddressesParser represents a MessageAddressesParser for the custom module
func CustomAddressesParser(_ codec.Codec, cosmosMsg sdk.Msg) ([]string, error) {
	switch msg := cosmosMsg.(type) {

	case *stakingtypes.MsgCancelUnbondingDelegation:
		return []string{msg.DelegatorAddress, msg.ValidatorAddress}, nil

	default:
		return nil, messages.MessageNotSupported(cosmosMsg)
	}
}
