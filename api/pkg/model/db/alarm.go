package db

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ego-component/egorm"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/clickvisual/clickvisual/api/internal/invoker"
)

type IAlarm interface {
	TableName() string
	RuleName(filterId int) string
	ViewName(database, table string, seq int) string
	UniqueName(filterId int) string
	StatusUpdate(status string) (err error)
	AlertInterval() string
}

const (
	_ int = iota
	AlarmModeAggregation
	AlarmModeAggregationCheck
)

const (
	AlarmStatusClose = iota + 1
	AlarmStatusOpen
	AlarmStatusFiring
	AlarmStatusRuleCheck
)

const (
	RuleStoreTypeFile         = 1
	RuleStoreTypeK8sConfigMap = 2
	RuleStoreTypeK8sOperator  = 3
)

var UnitMap = map[int]UnitItem{
	0: {
		Alias:    "m",
		Duration: time.Minute,
	},
	1: {
		Alias:    "s",
		Duration: time.Second,
	},
	2: {
		Alias:    "h",
		Duration: time.Hour,
	},
	3: {
		Alias:    "d",
		Duration: time.Hour * 24,
	},
	4: {
		Alias:    "w",
		Duration: time.Hour * 24 * 7,
	},
}

type UnitItem struct {
	Alias    string        `json:"alias"`
	Duration time.Duration `json:"duration"`
}

type RespAlarmListRelatedInfo struct {
	Table    BaseTable    `json:"table"`
	Instance BaseInstance `json:"instance"`
}

type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      time.Time         `json:"endsAt"`
}

type Notification struct {
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	Status            string            `json:"status"`
	Receiver          string            `json:"receiver"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Alerts            []Alert           `json:"alerts"`
}

type Alarm struct {
	BaseModel

	Uid        int           `gorm:"column:uid;type:int(11)" json:"uid"`                              // uid of alarm operator
	Uuid       string        `gorm:"column:uuid;type:varchar(128);NOT NULL" json:"uuid"`              // foreign key
	Name       string        `gorm:"column:name;type:varchar(128);NOT NULL" json:"alarmName"`         // name of an alarm
	Desc       string        `gorm:"column:desc;type:varchar(255);NOT NULL" json:"desc"`              // description
	Interval   int           `gorm:"column:interval;type:int(11)" json:"interval"`                    // interval second between alarm
	Unit       int           `gorm:"column:unit;type:int(11)" json:"unit"`                            // 0 m 1 s 2 h 3 d 4 w 5 y
	Tags       String2String `gorm:"column:tag;type:text" json:"tag"`                                 // tags
	Status     int           `gorm:"column:status;type:int(11)" json:"status"`                        // status
	ChannelIds Ints          `gorm:"column:channel_ids;type:varchar(255);NOT NULL" json:"channelIds"` // channel of an alarm
	NoDataOp   int           `gorm:"column:no_data_op;type:int(11)" db:"no_data_op" json:"noDataOp"`  // noDataOp 0 nodata 1 ok 2 alert
	Level      int           `gorm:"column:level;type:int(11)" json:"level"`                          // 0 m 1 s 2 h 3 d 4 w 5 y

	User *User `json:"user,omitempty" gorm:"foreignKey:uid;references:id"`

	// v2 field to support multiple alarm conditions
	ViewDDLs   String2String `gorm:"column:view_ddl_s;type:text" json:"viewDDLs"` // Users to store data generates the alarm condition
	TableIds   Ints          `gorm:"column:table_ids;type:varchar(255);NOT NULL" json:"tableIds"`
	AlertRules String2String `gorm:"column:alert_rules;type:text" json:"alertRules"` // prometheus alert rule

	// Deprecated: Tid
	Tid int `gorm:"column:tid;type:int(11)" json:"tid"` // table id
	// Deprecated: AlertRule will be replaced by AlertRules field, is expected to delete 0.5.0 version
	AlertRule string `gorm:"column:alert_rule;type:text" json:"alertRule"` // prometheus alert rule
	// Deprecated: View
	View string `gorm:"column:view;type:text" json:"view"` // view table ddl
	// Deprecated: ViewTableName
	ViewTableName string `gorm:"column:view_table_name;type:varchar(255)" json:"viewTableName"` // name of view table
}

func (m *Alarm) TableName() string {
	return TableNameAlarm
}

func (m *Alarm) RuleName(filterId int) string {
	if filterId == 0 {
		return fmt.Sprintf("cv-%s.yaml", m.Uuid)
	}
	return fmt.Sprintf("cv-%s-%d.yaml", m.Uuid, filterId)
}

func (m *Alarm) ViewName(database, table string, seq int) string {
	return fmt.Sprintf("%s.%s_%s", database, table, m.UniqueName(seq))
}

func (m *Alarm) UniqueName(filterId int) string {
	return strings.ReplaceAll(fmt.Sprintf("%s_%d", m.Uuid, filterId), "-", "_")
}

func (m *Alarm) AlertInterval() string {
	return fmt.Sprintf("%d%s", m.Interval, UnitMap[m.Unit].Alias)
}

func (m *Alarm) StatusUpdate(status string) (err error) {
	ups := make(map[string]interface{}, 0)
	if status == "firing" {
		ups["status"] = AlarmStatusFiring
	} else if status == "resolved" {
		ups["status"] = AlarmStatusOpen
	}
	err = AlarmUpdate(invoker.Db, m.ID, ups)
	if err != nil {
		return
	}
	return
}

// RuleNameMap 提供 rule 兼容
func (m *Alarm) RuleNameMap() map[int][]string {
	res := make(map[int][]string, 0)
	for iidRuleName := range m.AlertRules {
		iidTableArr := strings.Split(iidRuleName, "|")
		if len(iidTableArr) == 2 {
			iid, _ := strconv.Atoi(iidTableArr[0])
			res[iid] = append(res[iid], iidTableArr[1])
		}
	}

	return res
}

func GetAlarmTableInstanceInfo(id int) (alarmInfo Alarm, relatedList []*RespAlarmListRelatedInfo, err error) {
	alarmInfo, err = AlarmInfo(invoker.Db, id)
	if err != nil {
		return
	}
	relatedList = make([]*RespAlarmListRelatedInfo, 0)
	if len(alarmInfo.TableIds) != 0 {
		for _, tid := range alarmInfo.TableIds {
			var ri *RespAlarmListRelatedInfo
			ri, err = alarmRelatedInfo(tid)
			if err != nil {
				return
			}
			relatedList = append(relatedList, ri)
		}
		return
	}
	// TODO: wait delete
	var ri *RespAlarmListRelatedInfo
	ri, err = alarmRelatedInfo(alarmInfo.Tid)
	if err != nil {
		return
	}
	relatedList = append(relatedList, ri)
	return
}

func alarmRelatedInfo(tid int) (resp *RespAlarmListRelatedInfo, err error) {
	var (
		tableInfo    BaseTable
		instanceInfo BaseInstance
	)
	// table info
	tableInfo, err = TableInfo(invoker.Db, tid)
	if err != nil {
		return
	}
	// prometheus set
	instanceInfo, err = InstanceInfo(invoker.Db, tableInfo.Database.Iid)
	if err != nil {
		return
	}
	resp = &RespAlarmListRelatedInfo{
		Table:    tableInfo,
		Instance: instanceInfo,
	}
	return
}

func AlarmInfo(db *gorm.DB, id int) (resp Alarm, err error) {
	var sql = "`id`= ?"
	var binds = []interface{}{id}
	if err = db.Model(Alarm{}).Where(sql, binds...).First(&resp).Error; err != nil {
		err = errors.Wrapf(err, "alarm id: %d", id)
		return
	}
	return
}

func AlarmInfoX(db *gorm.DB, conds map[string]interface{}) (resp Alarm, err error) {
	sql, binds := egorm.BuildQuery(conds)
	if err = db.Model(Alarm{}).Where(sql, binds...).First(&resp).Error; err != nil && err != gorm.ErrRecordNotFound {
		err = errors.Wrapf(err, "conds: %v", conds)
		return
	}
	return
}

func AlarmList(conds egorm.Conds) (resp []*Alarm, err error) {
	sql, binds := egorm.BuildQuery(conds)
	if err = invoker.Db.Model(Alarm{}).Preload("User").Where(sql, binds...).Find(&resp).Error; err != nil {
		err = errors.Wrapf(err, "conds: %v", conds)
		return
	}
	return
}

func AlarmListByTidArr(conds egorm.Conds, tidArr []int) (resp []*Alarm, err error) {
	jcs := ""
	for _, tid := range tidArr {
		if jcs == "" {
			jcs = fmt.Sprintf("JSON_CONTAINS(`table_ids`, '[%d]')", tid)
			continue
		}
		jcs = fmt.Sprintf("%s OR JSON_CONTAINS(`table_ids`, '[%d]')", jcs, tid)
	}
	sql, binds := egorm.BuildQuery(conds)
	if err = invoker.Db.Model(Alarm{}).Preload("User").Where(sql, binds...).Where(jcs).Find(&resp).Error; err != nil {
		err = errors.Wrapf(err, "conds: %v", conds)
		return
	}
	return
}

// AlarmListPageByTidArr return item list by pagination
// SELECT *  FROM `cv_alarm` WHERE JSON_CONTAINS(`table_ids`, '[1]') OR JSON_CONTAINS(`table_ids`, '[7]')
func AlarmListPageByTidArr(conds egorm.Conds, reqList *ReqPage, tidArr []int) (total int64, respList []*Alarm) {
	respList = make([]*Alarm, 0)
	if reqList.PageSize == 0 {
		reqList.PageSize = 10
	}
	if reqList.Current == 0 {
		reqList.Current = 1
	}
	jcs := ""
	for _, tid := range tidArr {
		if jcs == "" {
			jcs = fmt.Sprintf("JSON_CONTAINS(`table_ids`, '[%d]')", tid)
			continue
		}
		jcs = fmt.Sprintf("%s OR JSON_CONTAINS(`table_ids`, '[%d]')", jcs, tid)
	}
	sql, binds := egorm.BuildQuery(conds)
	db := invoker.Db.Model(Alarm{}).Preload("User").Where(jcs).Where(sql, binds...).Order("id desc")
	db.Count(&total)
	db.Offset((reqList.Current - 1) * reqList.PageSize).Limit(reqList.PageSize).Find(&respList)
	return
}

func AlarmListByDidPage(conds egorm.Conds, reqList *ReqPage) (total int64, respList []*Alarm) {
	respList = make([]*Alarm, 0)
	if reqList.PageSize == 0 {
		reqList.PageSize = 10
	}
	if reqList.Current == 0 {
		reqList.Current = 1
	}
	sql, binds := egorm.BuildQuery(conds)
	db := invoker.Db.Select("cv_alarm.id").Model(Alarm{}).Preload("User").Joins("JOIN cv_base_table ON cv_base_table.id IN (replace(replace(JSON_EXTRACT(`cv_alarm`.`table_ids`, '$[*]'),'[',''),']',''))").Where(sql, binds...)
	db.Count(&total).Order("id desc")
	db.Offset((reqList.Current - 1) * reqList.PageSize).Limit(reqList.PageSize).Find(&respList)
	return
}

func AlarmCreate(db *gorm.DB, data *Alarm) (err error) {
	if err = db.Model(Alarm{}).Create(data).Error; err != nil {
		invoker.Logger.Error("create releaseZone error", zap.Error(err))
		return
	}
	return
}

func AlarmUpdate(db *gorm.DB, id int, ups map[string]interface{}) (err error) {
	var sql = "`id`=?"
	var binds = []interface{}{id}
	if err = db.Model(Alarm{}).Where(sql, binds...).Updates(ups).Error; err != nil {
		return errors.Wrapf(err, "ups: %v", ups)
	}
	return
}

func AlarmDelete(db *gorm.DB, id int) (err error) {
	if err = db.Model(Alarm{}).Unscoped().Delete(&Alarm{}, id).Error; err != nil {
		invoker.Logger.Error("release delete error", zap.Error(err))
		return
	}
	return
}

type ReqAlertSettingUpdate struct {
	RuleStoreType    int    `json:"ruleStoreType" form:"ruleStoreType"` // rule_store_type 1 文件 2 集群
	PrometheusTarget string `json:"prometheusTarget" form:"prometheusTarget"`

	// file
	FilePath string `json:"filePath" form:"filePath"`

	// k8s
	Namespace string `json:"namespace" form:"namespace"`
	Configmap string `json:"configmap" form:"configmap"`
	ClusterId int    `json:"clusterId" form:"clusterId"`
}

type RespAlertSettingInfo struct {
	InstanceId int `json:"instanceId"`
	ReqAlertSettingUpdate
}

type RespAlertSettingListItem struct {
	InstanceId       int    `json:"instanceId"`
	InstanceName     string `json:"instanceName"`
	RuleStoreType    int    `json:"ruleStoreType"` // rule_store_type 1 文件 2 集群
	PrometheusTarget string `json:"prometheusTarget"`

	// check
	IsPrometheusOK            int    `json:"isPrometheusOK"` // 0 no 1 yes
	CheckPrometheusResult     string `json:"checkPrometheusResult"`
	IsAlertManagerOK          int    `json:"isAlertManagerOK"` // 0 no 1 yes
	CheckAlertManagerResult   string `json:"checkAlertManagerResult"`
	IsMetricsSamplesOk        int    `json:"isMetricsSamplesOk"` // 0 no 1 yes
	CheckMetricsSamplesResult string `json:"checkMetricsSamplesResult"`
}

type ReqCreateMetricsSamples struct {
	Iid     int    `json:"iid" form:"iid"`
	Cluster string `json:"cluster" form:"cluster"`
}
