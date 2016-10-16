package models

import "testing"

var allowedBranches = [][]struct {
	branchId int
	access   int
}{
	{
		{1, 3},
		{2, 1},
		{7, 0},
		{2, 3},
	},
	// empty check
	{},
}

func TestEncodeUserPermission(t *testing.T) {
	for i, test := range allowedBranches {
		p := UserPermission{}
		for _, branch_authority := range test {
			p.Branches = append(p.Branches,
				BranchAccess{
					BranchId:  branch_authority.branchId,
					Access: branch_authority.access,
				})
		}

		decoded := UserPermission{}
		decoded.EncodedPermission = p.Encode()
		if err := decoded.Decode(); err != nil {
			t.Errorf("Decoding permission:(%d) failed, '%v'\n", i+1, err)
			continue
		}
		for j, branch_authority := range decoded.Branches {
			if branch_authority.BranchId != test[j].branchId ||
				branch_authority.Access != test[j].access {
				t.Errorf("Branch access(%d, %d) doesn't match, wanted %v got %v\n",
					i+1, j+1, test[j], branch_authority)
				continue
			}
		}
	}
}
