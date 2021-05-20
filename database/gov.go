package database

import (
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/gogo/protobuf/proto"

	"github.com/forbole/bdjuno/types"

	dbtypes "github.com/forbole/bdjuno/database/types"

	"github.com/lib/pq"
)

// SaveGovParams saves the given x/gov parameters inside the database
func (db *Db) SaveGovParams(params *types.GovParams) error {
	depositParamsBz, err := db.EncodingConfig.Marshaler.MarshalJSON(&params.DepositParams)
	if err != nil {
		return err
	}

	votingParamsBz, err := db.EncodingConfig.Marshaler.MarshalJSON(&params.VotingParams)
	if err != nil {
		return err
	}

	tallyingParams, err := db.EncodingConfig.Marshaler.MarshalJSON(&params.TallyParams)
	if err != nil {
		return err
	}

	stmt := `
INSERT INTO gov_params(deposit_params, voting_params, tally_params, height) 
VALUES ($1, $2, $3, $4)
ON CONFLICT (one_row_id) DO UPDATE 
	SET deposit_params = excluded.deposit_params,
  		voting_params = excluded.voting_params,
		tally_params = excluded.tally_params,
		height = excluded.height
WHERE gov_params.height <= excluded.height`
	_, err = db.Sql.Exec(stmt, string(depositParamsBz), string(votingParamsBz), string(tallyingParams), params.Height)
	return err
}

// GetGovParams returns the most recent governance parameters
func (db *Db) GetGovParams() (*types.GovParams, error) {
	var rows []dbtypes.GovParamsRow
	err := db.Sqlx.Select(&rows, `SELECT * FROM gov_params`)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, nil
	}

	row := rows[0]

	var depositParams govtypes.DepositParams
	err = db.EncodingConfig.Marshaler.UnmarshalJSON([]byte(row.DepositParams), &depositParams)
	if err != nil {
		return nil, err
	}

	var votingParams govtypes.VotingParams
	err = db.EncodingConfig.Marshaler.UnmarshalJSON([]byte(row.VotingParams), &votingParams)
	if err != nil {
		return nil, err
	}

	var tallyParams govtypes.TallyParams
	err = db.EncodingConfig.Marshaler.UnmarshalJSON([]byte(row.TallyParams), &tallyParams)
	if err != nil {
		return nil, err
	}

	return types.NewGovParams(
		govtypes.NewParams(votingParams, tallyParams, depositParams),
		row.Height,
	), nil
}

// --------------------------------------------------------------------------------------------------------------------

// SaveProposals allows to save for the given height the given total amount of coins
func (db *Db) SaveProposals(proposals []types.Proposal) error {
	if len(proposals) == 0 {
		return nil
	}

	query := `
INSERT INTO proposal(
	id, title, description, content, proposer_address, proposal_route, proposal_type, status, 
    submit_time, deposit_end_time, voting_start_time, voting_end_time
) VALUES`
	var param []interface{}
	for i, proposal := range proposals {
		vi := i * 12
		query += fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d),",
			vi+1, vi+2, vi+3, vi+4, vi+5, vi+6, vi+7, vi+8, vi+9, vi+10, vi+11, vi+12)

		// Encode the content properly
		protoContent, ok := proposal.Content.(proto.Message)
		if !ok {
			return fmt.Errorf("invalid proposal content types: %T", proposal.Content)
		}

		anyContent, err := codectypes.NewAnyWithValue(protoContent)
		if err != nil {
			return err
		}

		contentBz, err := db.EncodingConfig.Marshaler.MarshalJSON(anyContent)
		if err != nil {
			return err
		}

		param = append(param,
			proposal.ProposalID,
			proposal.Content.GetTitle(),
			proposal.Content.GetDescription(),
			string(contentBz),
			proposal.Proposer,
			proposal.ProposalRoute,
			proposal.ProposalType,
			proposal.Status,
			proposal.SubmitTime,
			proposal.DepositEndTime,
			proposal.VotingStartTime,
			proposal.VotingEndTime,
		)
	}
	query = query[:len(query)-1] // Remove trailing ","
	query += " ON CONFLICT DO NOTHING"
	_, err := db.Sql.Exec(query, param...)
	return err
}

// GetProposal returns the proposal with the given id, or nil if not found
func (db *Db) GetProposal(id uint64) (*types.Proposal, error) {
	var rows []*dbtypes.ProposalRow
	err := db.Sqlx.Select(&rows, `SELECT * FROM proposal WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, nil
	}

	row := rows[0]

	var contentAny codectypes.Any
	err = db.EncodingConfig.Marshaler.UnmarshalJSON([]byte(row.Content), &contentAny)
	if err != nil {
		return nil, err
	}

	var content govtypes.Content
	err = db.EncodingConfig.Marshaler.UnpackAny(&contentAny, &content)
	if err != nil {
		return nil, err
	}

	proposal := types.NewProposal(
		row.ProposalID,
		row.ProposalRoute,
		row.ProposalType,
		content,
		row.Status,
		row.SubmitTime,
		row.DepositEndTime,
		row.VotingStartTime,
		row.VotingEndTime,
		row.Proposer,
	)
	return &proposal, nil
}

// GetOpenProposalsIds returns all the ids of the proposals that are currently in deposit or voting period
func (db *Db) GetOpenProposalsIds() ([]uint64, error) {
	var ids []uint64
	stmt := `SELECT id FROM proposal WHERE status = $1 OR status = $2`
	err := db.Sqlx.Select(&ids, stmt, govtypes.StatusDepositPeriod.String(), govtypes.StatusVotingPeriod.String())
	return ids, err
}

// --------------------------------------------------------------------------------------------------------------------

// UpdateProposal updates a proposal stored inside the database
func (db *Db) UpdateProposal(update types.ProposalUpdate) error {
	query := `UPDATE proposal SET status = $1, voting_start_time = $2, voting_end_time = $3 where id = $4`
	_, err := db.Sql.Exec(query,
		update.Status,
		update.VotingStartTime,
		update.VotingEndTime,
		update.ProposalID,
	)
	return err
}

// SaveDeposits allows to save multiple deposits
func (db *Db) SaveDeposits(deposits []types.Deposit) error {
	if len(deposits) == 0 {
		return nil
	}

	query := `INSERT INTO proposal_deposit(proposal_id, depositor_address, amount, height) VALUES `
	var param []interface{}

	for i, deposit := range deposits {
		vi := i * 4
		query += fmt.Sprintf("($%d,$%d,$%d,$%d),", vi+1, vi+2, vi+3, vi+4)
		param = append(param, deposit.ProposalID,
			deposit.Depositor,
			pq.Array(dbtypes.NewDbCoins(deposit.Amount)),
			deposit.Height,
		)
	}
	query = query[:len(query)-1] // Remove trailing ","
	query += " ON CONFLICT DO NOTHING"
	_, err := db.Sql.Exec(query, param...)
	return err
}

// --------------------------------------------------------------------------------------------------------------------

// SaveVote allows to save for the given height and the message vote
func (db *Db) SaveVote(vote types.Vote) error {
	query := `INSERT INTO proposal_vote(proposal_id, voter_address, option, height) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`
	_, err := db.Sql.Exec(query,
		vote.ProposalID,
		vote.Voter,
		vote.Option.String(),
		vote.Height,
	)
	return err
}

// SaveTallyResults allows to save for the given height the given total amount of coins
func (db *Db) SaveTallyResults(tallys []types.TallyResult) error {
	if len(tallys) == 0 {
		return nil
	}
	query := `INSERT INTO proposal_tally_result(proposal_id, yes, abstain, no, no_with_veto, height) VALUES`
	var param []interface{}
	for i, tally := range tallys {
		vi := i * 6
		query += fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d),", vi+1, vi+2, vi+3, vi+4, vi+5, vi+6)
		param = append(param, tally.ProposalID,
			tally.Yes,
			tally.Abstain,
			tally.No,
			tally.NoWithVeto,
			tally.Height,
		)
	}
	query = query[:len(query)-1] // Remove trailing ","
	query += `
ON CONFLICT ON CONSTRAINT unique_tally_result DO UPDATE 
	SET yes = excluded.yes, 
	    abstain = excluded.abstain, 
	    no = excluded.no, 
	    no_with_veto = excluded.no_with_veto,
	    height = excluded.height
WHERE proposal_tally_result.height <= excluded.height`
	_, err := db.Sql.Exec(query, param...)
	return err
}