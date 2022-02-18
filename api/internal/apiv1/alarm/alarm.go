package alarm

import (
	"github.com/google/uuid"
	"github.com/gotomicro/ego-component/egorm"
	"github.com/spf13/cast"

	"github.com/shimohq/mogo/api/internal/invoker"
	"github.com/shimohq/mogo/api/pkg/component/core"
	"github.com/shimohq/mogo/api/pkg/model/db"
	"github.com/shimohq/mogo/api/pkg/model/view"
)

func Create(c *core.Context) {
	var req view.ReqAlarmCreate
	if err := c.Bind(&req); err != nil {
		c.JSONE(1, "invalid parameter: "+err.Error(), nil)
		return
	}
	tid := cast.ToInt(c.Param("id"))
	if tid == 0 {
		c.JSONE(core.CodeErr, "invalid parameter", nil)
		return
	}
	tx := invoker.Db.Begin()
	obj := &db.Alarm{
		Tid:      tid,
		Uuid:     uuid.NewString(),
		Name:     req.Name,
		Desc:     req.Desc,
		Interval: req.Interval,
		Unit:     req.Unit,
	}
	err := db.AlarmCreate(tx, obj)
	if err != nil {
		tx.Rollback()
		c.JSONE(1, "create failed: "+err.Error(), nil)
		return
	}
	for _, filter := range req.Filters {
		filterObj := &db.AlarmFilter{
			AlarmId:        obj.ID,
			When:           filter.When,
			SetOperatorTyp: filter.SetOperatorTyp,
			SetOperatorExp: filter.SetOperatorExp,
		}
		err = db.AlarmFilterCreate(tx, filterObj)
		if err != nil {
			tx.Rollback()
			c.JSONE(1, "create failed: "+err.Error(), nil)
			return
		}
	}
	for _, condition := range req.Conditions {
		conditionObj := &db.AlarmCondition{
			AlarmId: obj.ID,
			Exp:     condition.Exp,
			Cond:    condition.Cond,
			Val1:    condition.Val1,
			Val2:    condition.Val2,
		}
		err = db.AlarmConditionCreate(tx, conditionObj)
		if err != nil {
			tx.Rollback()
			c.JSONE(1, "create failed: "+err.Error(), nil)
			return
		}
	}
	if err = tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSONE(1, "create failed: "+err.Error(), nil)
		return
	}
	c.JSONOK()
}

func Update(c *core.Context) {
	id := cast.ToInt(c.Param("id"))
	if id == 0 {
		c.JSONE(1, "invalid parameter", nil)
		return
	}
	var req view.ReqAlarmCreate
	if err := c.Bind(&req); err != nil {
		c.JSONE(1, "invalid parameter: "+err.Error(), nil)
		return
	}
	tx := invoker.Db.Begin()
	ups := make(map[string]interface{}, 0)
	ups["name"] = req.Name
	ups["desc"] = req.Desc
	ups["interval"] = req.Interval
	ups["unit"] = req.Unit
	if err := db.AlarmUpdate(tx, id, ups); err != nil {
		tx.Rollback()
		c.JSONE(1, "update failed: "+err.Error(), nil)
		return
	}
	// filter
	if err := db.AlarmFilterDeleteBatch(tx, id); err != nil {
		tx.Rollback()
		c.JSONE(1, "update failed: "+err.Error(), nil)
		return
	}
	for _, filter := range req.Filters {
		filterObj := &db.AlarmFilter{
			AlarmId:        id,
			When:           filter.When,
			SetOperatorTyp: filter.SetOperatorTyp,
			SetOperatorExp: filter.SetOperatorExp,
		}
		if err := db.AlarmFilterCreate(tx, filterObj); err != nil {
			tx.Rollback()
			c.JSONE(1, "create failed: "+err.Error(), nil)
			return
		}
	}
	// condition
	if err := db.AlarmConditionDeleteBatch(tx, id); err != nil {
		tx.Rollback()
		c.JSONE(1, "update failed: "+err.Error(), nil)
		return
	}
	for _, condition := range req.Conditions {
		conditionObj := &db.AlarmCondition{
			AlarmId: id,
			Exp:     condition.Exp,
			Cond:    condition.Cond,
			Val1:    condition.Val1,
			Val2:    condition.Val2,
		}
		if err := db.AlarmConditionCreate(tx, conditionObj); err != nil {
			tx.Rollback()
			c.JSONE(1, "create failed: "+err.Error(), nil)
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSONE(1, "create failed: "+err.Error(), nil)
		return
	}
	c.JSONOK()
}

func List(c *core.Context) {
	res, err := db.AlarmList(egorm.Conds{})
	if err != nil {
		c.JSONE(core.CodeErr, err.Error(), nil)
		return
	}
	c.JSONE(core.CodeOK, "succ", res)
	return
}

func Info(c *core.Context) {
	id := cast.ToInt(c.Param("id"))
	if id == 0 {
		c.JSONE(1, "invalid parameter", nil)
		return
	}
	alarmInfo, err := db.AlarmInfo(invoker.Db, id)
	if err != nil {
		c.JSONE(core.CodeErr, err.Error(), nil)
		return
	}
	conds := egorm.Conds{}
	conds["alarm_id"] = alarmInfo.ID
	filters, err := db.AlarmFilterList(conds)
	if err != nil {
		c.JSONE(core.CodeErr, err.Error(), nil)
		return
	}
	conditions, err := db.AlarmConditionList(conds)
	if err != nil {
		c.JSONE(core.CodeErr, err.Error(), nil)
		return
	}
	res := view.ReqAlarmInfo{
		Alarm:      alarmInfo,
		Filters:    filters,
		Conditions: conditions,
	}
	c.JSONE(core.CodeOK, "succ", res)
	return
}

func Delete(c *core.Context) {
	id := cast.ToInt(c.Param("id"))
	if id == 0 {
		c.JSONE(1, "invalid parameter", nil)
		return
	}
	tx := invoker.Db.Begin()
	if err := db.AlarmDelete(tx, id); err != nil {
		c.JSONE(1, "failed to delete: "+err.Error(), nil)
		return
	}
	// filter
	if err := db.AlarmFilterDeleteBatch(tx, id); err != nil {
		tx.Rollback()
		c.JSONE(1, "update failed: "+err.Error(), nil)
		return
	}
	// condition
	if err := db.AlarmConditionDeleteBatch(tx, id); err != nil {
		tx.Rollback()
		c.JSONE(1, "update failed: "+err.Error(), nil)
		return
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSONE(1, "create failed: "+err.Error(), nil)
		return
	}
	c.JSONOK()
}
