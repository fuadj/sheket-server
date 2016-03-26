package models

import (
	"database/sql"
	"fmt"
)

type ShEntityRevision struct {
	CompanyId        int64
	RevisionNumber   int64
	// The type of element this revision applies,
	// e.g:(transaction, items, ...)
	EntityType       int64
	// The action that caused the revision # to change
	// e.g:(insert, update, ...)
	ActionType       int64
	// The entity id affected by the change
	EntityAffectedId int64
	// Any other info necessary
	AdditionalInfo   int64
}

const (
	REV_ACTION_CREATE = iota + 1
	REV_ACTION_UPDATE
	REV_ACTION_DELETE

	REV_ENTITY_ITEM = iota + 1
	REV_ENTITY_BRANCH
	REV_ENTITY_BRANCH_ITEM
)

func (s *shStore) AddEntityRevisionInTx(tnx *sql.Tx, rev *ShEntityRevision) (*ShEntityRevision, error) {
	rows, err := tnx.Query(
		fmt.Sprintf("select max(revision_number) from %s "+
			"where company_id = $1 and entity_type = $2", TABLE_ENTITY_REVISION),
		rev.CompanyId, rev.EntityType)
	if err != nil {
		return nil, fmt.Errorf("can't query rev # for entity:%d, %v", rev.EntityType,
			err)
	}

	max_rev := int64(0)
	if rows.Next() {
		err = rows.Scan(&max_rev)
		if err != nil {
			return nil, err
		}
	}

	max_rev++

	_, err = tnx.Exec(
		fmt.Sprintf("insert into %s "+
			"company_id, revision_number, entity_type, action_type, "+
			"affected_id, additional_info values "+
			"($1, $2, $3, $4, $5, $6)", TABLE_ENTITY_REVISION),
		rev.CompanyId, max_rev, rev.EntityType, rev.ActionType,
		rev.EntityAffectedId, rev.AdditionalInfo)
	if err != nil {
		return nil, err
	}
	return rev, nil
}

func (s *shStore) GetRevisionsSince(prev_rev *ShEntityRevision) (int64, []*ShEntityRevision, error) {
	var result []*ShEntityRevision

	var max_rev int64 = prev_rev.RevisionNumber

	rows, err := s.Query(
		fmt.Sprintf("select "+
			"company_id, revision_number, entity_type, action_type, "+
			"affected_id, additional_info from %s "+
			"where revision_number > $1 group by affected_id "+
			"order by revision_number asc", TABLE_ENTITY_REVISION),
		prev_rev.RevisionNumber)
	if err != nil {
		return max_rev, nil, err
	}

	for rows.Next() {
		rev := new(ShEntityRevision)
		err := rows.Scan(
			&rev.CompanyId,
			&rev.RevisionNumber,
			&rev.EntityType,
			&rev.ActionType,
			&rev.EntityAffectedId,
			&rev.AdditionalInfo,
		)
		if err != nil {
			return max_rev, nil, err
		}

		if max_rev < rev.RevisionNumber {
			max_rev = rev.RevisionNumber
		}
		result = append(result, rev)
	}

	return max_rev, result, nil
}

