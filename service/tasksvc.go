package service

import (
	"context"
	"sync"

	"github.com/galaxy-future/schedulx/api/types"
	"github.com/galaxy-future/schedulx/pkg/tool"
	"github.com/galaxy-future/schedulx/register/config/log"
	"github.com/galaxy-future/schedulx/repository"
	"github.com/galaxy-future/schedulx/repository/model/db"
)

type TaskService struct {
}

var taskServiceInstance *TaskService
var taskOnce sync.Once

func GetTaskSvcInst() *TaskService {
	taskOnce.Do(func() {
		taskServiceInstance = &TaskService{}
	})
	return taskServiceInstance
}

func (s *TaskService) entryLog(ctx context.Context, method string, req interface{}) {
	log.Logger.Infof("entry log | method[%s] | req:%s", method, tool.ToJson(req))
}

func (s *TaskService) exitLog(ctx context.Context, method string, req, resp interface{}, err error) {
	log.Logger.Infof("exit log | method[%s] | req:%s | resp:%s | err:%v", method, tool.ToJson(req), tool.ToJson(resp), err)
}

func (s *TaskService) Describe(ctx context.Context, svcReq *TaskDescribeSvcReq) (*TaskDescribeSvcResp, error) {
	var err error
	resp := &TaskDescribeSvcResp{
		TaskDescribe: &types.TaskDescribe{},
	}
	s.entryLog(ctx, "Describe", svcReq) // todo 日志脱敏
	defer func() {
		s.exitLog(ctx, "Describe", svcReq, resp, err)
	}()
	repo := repository.GetInstanceRepoIns()
	fields := []string{
		"id",
		"instance_status",
		"ip_inner",
	}
	insts, err := repo.InstsQueryByTaskId(ctx, svcReq.TaskId, "", fields) //todo 改为查 nodeact_task 表
	if err != nil {
		log.Logger.Error("InstsQueryByTaskId", err)
		return nil, err
	}
	if len(insts) != 0 {
		row := &db.Instance{}
		err = db.Get(insts[0].Id, row)
		if err != nil {
			log.Logger.Error("db Get", err)
			return nil, err
		}
		resp.TaskDescribe.FoundTime = row.CreateAt
	}

	var successNum, failNum int64
	for _, inst := range insts {
		if inst.InstanceStatus == svcReq.InstanceStatus {
			successNum++
			continue
		}
		if inst.InstanceStatus == types.InstanceStatusFail {
			failNum++
			continue
		}
	}
	totalNum := int64(len(insts))
	resp.TaskDescribe.FailNum = failNum
	resp.TaskDescribe.SuccessNum = successNum
	resp.TaskDescribe.TotalNum = totalNum
	resp.TaskDescribe.SuccessRate = tool.FormatFloat(float64(successNum/totalNum), 2)
	return resp, nil
}

func (s *TaskService) Instances(ctx context.Context, svcReq *TaskInstancesSvcReq) (*TaskInstancesSvcResp, error) {
	var err error
	resp := &TaskInstancesSvcResp{}
	s.entryLog(ctx, "Instances", svcReq) // todo 日志脱敏
	defer func() {
		s.exitLog(ctx, "Instances", svcReq, resp, err)
	}()
	repo := repository.GetInstanceRepoIns()
	fields := []string{
		"instance_id",
		"ip_inner",
		"ip_outer",
	}
	insts, count, err := repo.InstsQueryByPage(ctx, svcReq.TaskId, svcReq.InstanceStatus, svcReq.PageSize, svcReq.PageNumber, fields)
	if err != nil {
		log.Logger.Error("InstsQueryByTaskId", err)
		return nil, err
	}
	instances := make([]*types.Instance, 0, len(insts))
	for _, item := range insts {
		ins := &types.Instance{
			InstanceId: item.InstanceId,
			IpInner:    item.IpInner,
			IpOuter:    item.IpOuter,
			Status:     svcReq.InstanceStatus,
		}
		instances = append(instances, ins)
	}
	pager := &types.Pager{
		PagerNum:  svcReq.PageNumber,
		PagerSize: svcReq.PageSize,
		Total:     int(count),
	}
	resp.InstancesList = instances
	resp.Pager = pager
	return resp, nil
}