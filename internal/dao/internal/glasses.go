// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// GlassesDao is the data access object for the table glasses.
type GlassesDao struct {
	table    string             // table is the underlying table name of the DAO.
	group    string             // group is the database configuration group name of the current DAO.
	columns  GlassesColumns     // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler // handlers for customized model modification.
}

// GlassesColumns defines and stores column names for the table glasses.
type GlassesColumns struct {
	Id                          string //
	Name                        string // 设备名称
	BatteryLevel                string // 电量
	SystemVersion               string // 系统版本
	EquipmentModel              string // 设备型号
	EquipmentSn                 string // 设备序列号
	CustomerSn                  string // 客户序列号
	InternalStorageSpace        string // 设备存储
	BluetoothStatus             string // 蓝牙状态
	BluetoothName               string // 蓝牙名称
	BluetoothMacAddress         string // 蓝牙mac地址
	WifiStatus                  string // wifi状态
	WifiNameConnected           string // 已连接的wifi状态
	WlanMacAddress              string // wlan mac 地址
	DeviceIpAddress             string // 设备 ip 地址
	ChargingStatus              string // 设备充电状态
	BluetoothInfDevice          string // 有关设备原始蓝牙信息
	BluetoothInfConnected       string // 已连接蓝牙信息
	CameraTemperatureCelsius    string // 相机的温度，摄氏度
	CameraTemperatureFahrenheit string // 相机温度，华氏度
	LargespaceMapInfo           string // 大空间地图信息
	Trackers                    string // 追踪器
	Status                      string // 1 上线 2 下线
	CreatedAt                   string //
	UpdatedAt                   string //
}

// glassesColumns holds the columns for the table glasses.
var glassesColumns = GlassesColumns{
	Id:                          "id",
	Name:                        "name",
	BatteryLevel:                "battery_level",
	SystemVersion:               "system_version",
	EquipmentModel:              "equipment_model",
	EquipmentSn:                 "equipment_sn",
	CustomerSn:                  "customer_sn",
	InternalStorageSpace:        "internal_storage_space",
	BluetoothStatus:             "bluetooth_status",
	BluetoothName:               "bluetooth_name",
	BluetoothMacAddress:         "bluetooth_mac_address",
	WifiStatus:                  "wifi_status",
	WifiNameConnected:           "wifi_name_connected",
	WlanMacAddress:              "wlan_mac_address",
	DeviceIpAddress:             "device_ip_address",
	ChargingStatus:              "charging_status",
	BluetoothInfDevice:          "bluetooth_inf_device",
	BluetoothInfConnected:       "bluetooth_inf_connected",
	CameraTemperatureCelsius:    "camera_temperature_celsius",
	CameraTemperatureFahrenheit: "camera_temperature_fahrenheit",
	LargespaceMapInfo:           "largespace_map_info",
	Trackers:                    "trackers",
	Status:                      "status",
	CreatedAt:                   "created_at",
	UpdatedAt:                   "updated_at",
}

// NewGlassesDao creates and returns a new DAO object for table data access.
func NewGlassesDao(handlers ...gdb.ModelHandler) *GlassesDao {
	return &GlassesDao{
		group:    "default",
		table:    "glasses",
		columns:  glassesColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *GlassesDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *GlassesDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *GlassesDao) Columns() GlassesColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *GlassesDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *GlassesDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rolls back the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
//
// Note: Do not commit or roll back the transaction in function f,
// as it is automatically handled by this function.
func (dao *GlassesDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
