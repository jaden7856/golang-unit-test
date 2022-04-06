package policy

import (
	"10.1.1.220/cdm/cdm-cloud/common/database"
	"10.1.1.220/cdm/cdm-cloud/common/errors"
	"10.1.1.220/cdm/cdm-cloud/common/test/helper"
	"10.1.1.220/cdm/cdm-replicator/common/database/model"
	"10.1.1.220/cdm/cdm-replicator/services/policy/proto"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"

	"os"
	"testing"
	"time"
)

var (
	stringPtr = func(s string) *string { return &s }
)

func panicIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func fatalIfError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestMain(m *testing.M) {
	panicIfError(helper.Init(
		// helper 는 따로 go-micro 에서 커스터마이징한 함수
		helper.DatabaseDDLScriptURI("test.ddl"),
	))
	defer helper.Close()

	time.Sleep(2 * time.Second)

	if code := m.Run(); code != 0 {
		os.Exit(code)
	}
}

func TestGetNodeList(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		var pns []*model.PolicyNode
		pns = append(pns, &model.Test{
			Name:    "test1",
			Ip:      "1.1.1.1",
			Port:    5555,
			Remarks: stringPtr("remarks1"),
		})
		pns = append(pns, &model.Test{
			Name:    "test2",
			Ip:      "2.2.2.2",
			Port:    5555,
			Remarks: stringPtr("remarks2"),
		})
		pns = append(pns, &model.Test{
			Name:    "test3",
			Ip:      "3.3.3.3",
			Port:    5555,
			Remarks: stringPtr("remarks3"),
		})
		pns = append(pns, &model.Test{
			Name:    "diff1",
			Ip:      "4.4.4.4",
			Port:    5555,
			Remarks: stringPtr("remarks4"),
		})
		for _, pn := range pns {
			fatalIfError(t, db.Save(&pn).Error)
		}

		for _, tc := range []struct {
			Case     string
			Limit    uint64
			Offset   uint64
			Name     string
			Expected int
			Error    string
		}{
			{
				Case:     "no filter",
				Expected: 4,
			},
			{
				Case:     "name filter",
				Name:     "test",
				Expected: 3,
			},
			{
				Case:     "pagination filter",
				Offset:   2,
				Limit:    2,
				Expected: 2,
			},
			{
				Case:     "unknown filter data",
				Name:     "unknown",
				Expected: 0,
			},
			{
				Case:  "abnormal case: name length overflow",
				Name:  generateString(51),
				Error: "length overflow parameter value",
			},
		} {
			rsp, _, err := GetList(db, &proto.RequestGetList{
				Limit:  &wrappers.UInt64Value{Value: tc.Limit},
				Offset: &wrappers.UInt64Value{Value: tc.Offset},
				Name:   tc.Name,
			})
			if tc.Error != "" {
				assert.EqualError(t, err, tc.Error)
			} else {
				assert.NoError(t, err, tc.Case)
				assert.Equal(t, tc.Expected, len(rsp), tc.Case)
			}
		}
	})
}

func TestGetNodeDetail(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		pn := model.Test{
			Name:    "test",
			Ip:      "1.1.1.1",
			Port:    5555,
			Remarks: stringPtr("remarks"),
		}
		fatalIfError(t, db.Save(&pn).Error)

		for _, tc := range []struct {
			Case   string
			NodeID uint64
			Error  error
		}{
			{
				Case:   "Normal case",
				NodeID: pn.ID,
				Error:  nil,
			},
			{
				Case:   "Abnormal case",
				NodeID: 0,
				Error:  errors.ErrRequiredParameter,
			},
		} {
			rsp, err := Get(db, &proto.RequestGet{Id: tc.NodeID})
			if err != nil {
				assert.Error(t, err, tc.Case)
				assert.Equal(t, tc.Error.Error(), err.Error(), tc.Case)
			} else {
				assert.NoError(t, err)
				// Node
				assert.Equal(t, pn.ID, rsp.Id)
				assert.Equal(t, pn.Name, rsp.Name)
				assert.Equal(t, pn.Ip, rsp.Ip)
				assert.Equal(t, pn.Port, rsp.Port)
				assert.Equal(t, *pn.Remarks, rsp.Remarks)
			}
		}
	})
}

func TestAddNode(t *testing.T) {
	database.Test(func(db *gorm.DB) {
		//// abnormal case1 : no node property
		_, err := Add(db, &proto.RequestAdd{})
		assert.EqualError(t, err, "required parameter")

		rsp, err := Add(db, &proto.RequestAddNode{Node: &proto.Node{
			Name:    "test",
			Ip:      "1.1.1.1",
			Port:    5555,
			Remarks: "test",
		}})
		assert.NoError(t, err)
		// Node
		assert.Equal(t, "test", rsp.Name)
		assert.Equal(t, "1.1.1.1", rsp.Ip)
		assert.Equal(t, uint64(5555), rsp.Port)
		assert.Equal(t, "test", rsp.Remarks)
	})
}

func generateString(len int) string {
	var s string
	for i := 0; i < len; i++ {
		s += "a"
	}
	return s
}
