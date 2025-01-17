package db

import (
	"fmt"

	"github.com/ego-component/egorm"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/clickvisual/clickvisual/api/internal/invoker"
)

// BaseInstance 服务配置存储
type BaseInstance struct {
	BaseModel

	Datasource       string  `gorm:"column:datasource;type:varchar(32);NOT NULL;index:idx_datasource_name,unique" json:"datasource"` // datasource type
	Name             string  `gorm:"column:name;type:varchar(128);NOT NULL;index:idx_datasource_name,unique" json:"name"`            // datasource instance name
	Dsn              string  `gorm:"column:dsn;type:text" json:"dsn"`                                                                // dsn
	RuleStoreType    int     `gorm:"column:rule_store_type;type:int(11)" json:"ruleStoreType"`                                       // rule_store_type 1 文件 2 集群
	FilePath         string  `gorm:"column:file_path;type:varchar(255)" json:"filePath"`                                             // file_path
	Desc             string  `gorm:"column:desc;type:varchar(255)" json:"desc"`                                                      // file_path
	ClusterId        int     `gorm:"column:cluster_id;type:int(11)" json:"clusterId"`                                                // cluster_id
	Namespace        string  `gorm:"column:namespace;type:varchar(128)" json:"namespace"`                                            // namespace
	Configmap        string  `gorm:"column:configmap;type:varchar(128)" json:"configmap"`                                            // configmap
	PrometheusTarget string  `gorm:"column:prometheus_target;type:varchar(128)" json:"prometheusTarget"`                             // prometheus ip or domain, eg: https://prometheus:9090
	Mode             int     `gorm:"column:mode;type:tinyint(1)" json:"mode"`                                                        // 0 standalone 1 cluster
	ReplicaStatus    int     `gorm:"column:replica_status;type:tinyint(1)" json:"replicaStatus"`                                     // status 0 has replica 1 no replica
	Clusters         Strings `gorm:"column:clusters;type:text" json:"clusters"`
}

func (b *BaseInstance) TableName() string {
	return TableNameBaseInstance
}

func (b *BaseInstance) DsKey() string {
	return InstanceKey(b.ID)
}

func InstanceKey(id int) string {
	return fmt.Sprintf("%d", id)
}

// InstanceList ..
func InstanceList(conds egorm.Conds, extra ...string) (resp []*BaseInstance, err error) {
	sql, binds := egorm.BuildQuery(conds)
	sorts := ""
	if len(extra) >= 1 {
		sorts = extra[0]
	}
	if sorts == "" {
		sorts = "id desc"
	}
	if err = invoker.Db.Model(BaseInstance{}).Where(sql, binds...).Order(sorts).Find(&resp).Error; err != nil {
		err = errors.Wrapf(err, "conds: %v", conds)
		return
	}
	return
}

func InstanceCreate(db *gorm.DB, data *BaseInstance) (err error) {
	if err = db.Model(BaseInstance{}).Create(data).Error; err != nil {
		invoker.Logger.Error("create release error", zap.Error(err))
		return
	}
	return
}

func InstanceInfo(db *gorm.DB, id int) (resp BaseInstance, err error) {
	var sql = "`id`= ?"
	var binds = []interface{}{id}
	if err = db.Model(BaseInstance{}).Where(sql, binds...).First(&resp).Error; err != nil {
		return resp, errors.Wrapf(err, "instance id: %d", id)
	}
	return
}

func InstanceDelete(db *gorm.DB, id int) (err error) {
	if err = db.Model(BaseInstance{}).Unscoped().Delete(&BaseInstance{}, id).Error; err != nil {
		invoker.Logger.Error("release delete error", zap.Error(err))
		return
	}
	return
}

func InstanceUpdate(db *gorm.DB, id int, ups map[string]interface{}) (err error) {
	var sql = "`id`=?"
	var binds = []interface{}{id}
	if err = db.Model(BaseInstance{}).Where(sql, binds...).Updates(ups).Error; err != nil {
		invoker.Logger.Error("release update error", zap.Error(err))
		return
	}
	return
}

// InstanceInfoX Info extension method to query a single record according to Cond
func InstanceInfoX(db *gorm.DB, conds map[string]interface{}) (resp BaseInstance, err error) {
	sql, binds := egorm.BuildQuery(conds)
	if err = db.Table(TableNameBaseInstance).Where(sql, binds...).First(&resp).Error; err != nil && err != gorm.ErrRecordNotFound {
		invoker.Logger.Error("infoX error", zap.Error(err))
		return
	}
	return
}
