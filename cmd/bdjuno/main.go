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

	initcmd "github.com/forbole/juno/v4/cmd/init"
	parsetypes "github.com/forbole/juno/v4/cmd/parse/types"
	startcmd "github.com/forbole/juno/v4/cmd/start"
	"github.com/forbole/juno/v4/modules/messages"

	migratecmd "github.com/forbole/bdjuno/v4/cmd/migrate"
	parsecmd "github.com/forbole/bdjuno/v4/cmd/parse"
	"github.com/forbole/bdjuno/v4/database"
	"github.com/forbole/bdjuno/v4/modules"
	"github.com/forbole/bdjuno/v4/types/config"
	"github.com/forbole/juno/v4/cmd"
)

func main() {
	initCfg := initcmd.NewConfig().
		WithConfigCreator(config.Creator)

	parseCfg := parsetypes.NewConfig().
		WithDBBuilder(database.Builder).
		WithEncodingConfigBuilder(config.MakeEncodingConfig(getBasicManagers())).
		WithRegistrar(modules.NewRegistrar(getAddressesParser()))

	cfg := cmd.NewConfig("bdjuno").
		WithInitConfig(initCfg).
		WithParseConfig(parseCfg)

	// Run the command
	rootCmd := cmd.RootCmd(cfg.GetName())

	rootCmd.AddCommand(
		cmd.VersionCmd(),
		initcmd.NewInitCmd(cfg.GetInitConfig()),
		parsecmd.NewParseCmd(cfg.GetParseConfig()),
		migratecmd.NewMigrateCmd(cfg.GetName(), cfg.GetParseConfig()),
		startcmd.NewStartCmd(cfg.GetParseConfig()),
	)

	executor := cmd.PrepareRootCmd(cfg.GetName(), rootCmd)
	err := executor.Execute()
	if err != nil {
		panic(err)
	}
}

// getBasicManagers returns the various basic managers that are used to register the encoding to
// support custom messages.
// This should be edited by custom implementations if needed.
func getBasicManagers() []module.BasicManager {
	return []module.BasicManager{
		module.NewBasicManager(
			auth.AppModuleBasic{},
			//authzmodule.AppModuleBasic{},
			genutil.AppModuleBasic{},
			bank.AppModuleBasic{},
			//capability.AppModuleBasic{},
			staking.AppModuleBasic{},
			//mint.AppModuleBasic{},
			//distr.AppModuleBasic{},
			gov.NewAppModuleBasic(getGovProposalHandlers()),
			params.AppModuleBasic{},
			//crisis.AppModuleBasic{},
			slashing.AppModuleBasic{},
			//feegrantmodule.AppModuleBasic{},
			//groupmodule.AppModuleBasic{},
			//ibc.AppModuleBasic{},
			upgrade.AppModuleBasic{},
			//evidence.AppModuleBasic{},
			//transfer.AppModuleBasic{},
			//ica.AppModuleBasic{},
			vesting.AppModuleBasic{},
		),
	}
}

func getGovProposalHandlers() []govclient.ProposalHandler {
	var govProposalHandlers []govclient.ProposalHandler

	govProposalHandlers = append(govProposalHandlers,
		paramsclient.ProposalHandler,
		//distrclient.ProposalHandler,
		upgradeclient.LegacyProposalHandler,
		upgradeclient.LegacyCancelProposalHandler,
		//ibcclientclient.UpdateClientProposalHandler,
		//ibcclientclient.UpgradeProposalHandler,
	)

	return govProposalHandlers
}

// getAddressesParser returns the messages parser that should be used to get the users involved in
// a specific message.
// This should be edited by custom implementations if needed.
func getAddressesParser() messages.MessageAddressesParser {
	return messages.JoinMessageParsers(
		MissingStakingMessagesParser,
		messages.CosmosMessageAddressesParser,
	)
}

func MissingStakingMessagesParser(_ codec.Codec, cosmosMsg sdk.Msg) ([]string, error) {
	switch msg := cosmosMsg.(type) {

	case *stakingtypes.MsgCancelUnbondingDelegation:
		return []string{msg.DelegatorAddress, msg.ValidatorAddress}, nil

	}

	return nil, messages.MessageNotSupported(cosmosMsg)
}
