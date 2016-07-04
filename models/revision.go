package models

import (
	"database/sql"
	"fmt"
)

type ShEntityRevision struct {
	CompanyId      int64
	RevisionNumber int64
	// The type of element this revision applies,
	// e.g:(transaction, items, ...)
	EntityType int64
	// The action that caused the revision # to change
	// e.g:(insert, update, ...)
	ActionType int64
	// The entity id affected by the change
	EntityAffectedId int64
	// Any other info necessary
	AdditionalInfo int64
}

const (
	REV_ACTION_CREATE int64 = 1
	REV_ACTION_UPDATE int64 = 2
	REV_ACTION_DELETE int64 = 3
)

const (
	REV_ENTITY_ITEM        int64 = 1
	REV_ENTITY_BRANCH      int64 = 2
	REV_ENTITY_BRANCH_ITEM int64 = 3
	REV_ENTITY_MEMBERS     int64 = 4
	REV_ENTITY_CATEGORY    int64 = 5
)

func (s *shStore) AddEntityRevisionInTx(tnx *sql.Tx, rev *ShEntityRevision) (*ShEntityRevision, error) {
	rows, err := tnx.Query(
		fmt.Sprintf("select MAX(revision_number) from %s "+
			"where company_id = $1 and entity_type = $2", TABLE_ENTITY_REVISION),
		rev.CompanyId, rev.EntityType)
	if err != nil {
		return nil, fmt.Errorf("can't query rev # for entity:%d, %v", rev.EntityType,
			err)
	}

	max_rev := int64(0)
	var i sql.NullInt64
	if rows.Next() {
		err = rows.Scan(&i)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("Revision Scan error : %s", err.Error())
		}
		if i.Valid {
			max_rev = i.Int64
		}
	}
	// we don't defer the rows.Close b/c we want to release the connection
	// so that the Exec statement can use it immediately
	rows.Close()

	max_rev++

	_, err = tnx.Exec(
		fmt.Sprintf("insert into %s "+
			"(company_id, revision_number, entity_type, action_type, "+
			"affected_id, additional_info) values "+
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

	rows, err := s.Query(
		fmt.Sprintf("select distinct on (affected_id, additional_info) "+
			"company_id, revision_number, entity_type, action_type, "+
			"affected_id, additional_info from %s "+
			"where company_id = $1 AND entity_type = $2 AND "+
			"revision_number > $3 ",
			TABLE_ENTITY_REVISION),
		prev_rev.CompanyId, prev_rev.EntityType, prev_rev.RevisionNumber)
	if err != nil {
		return prev_rev.RevisionNumber, nil, fmt.Errorf("Revision query error : %s", err.Error())
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
		if err == sql.ErrNoRows {
			// no-op
		} else if err != nil {
			rows.Close()
			return prev_rev.RevisionNumber, nil, fmt.Errorf("Revision Scan error : %s", err.Error())
		} else {
			result = append(result, rev)
		}
	}
	// we don't defer the rows.Close b/c we want to release the connection
	// so that the Exec statement can use it immediately
	rows.Close()

	rows, err = s.Query(
		fmt.Sprintf("select max(revision_number) from %s "+
			"where company_id = $1 AND entity_type = $2", TABLE_ENTITY_REVISION),
		prev_rev.CompanyId, prev_rev.EntityType)
	if err != nil {
		return prev_rev.RevisionNumber, nil, fmt.Errorf("Revision query error : %s", err.Error())
	}

	defer rows.Close()

	max_rev := prev_rev.RevisionNumber
	if rows.Next() {
		var i sql.NullInt64
		if rows.Scan(&i) == nil {
			max_rev = i.Int64
		}
	}

	return max_rev, result, nil
}
