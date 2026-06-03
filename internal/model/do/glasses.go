// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// Glasses is the golang structure of table glasses for DAO operations like Where/Data.
type Glasses struct {
	g.Meta                      `orm:"table:glasses, do:true"`
	Id                          any         //
	Name                        any         // 设备名称
	BatteryLevel                any         // 电量
	SystemVersion               any         // 系统版本
	EquipmentModel              any         // 设备型号
	EquipmentSn                 any         // 设备序列号
	CustomerSn                  any         // 客户序列号
	InternalStorageSpace        any         // 设备存储
	BluetoothStatus             any         // 蓝牙状态
	BluetoothName               any         // 蓝牙名称
	BluetoothMacAddress         any         // 蓝牙mac地址
	WifiStatus                  any         // wifi状态
	WifiNameConnected           any         // 已连接的wifi状态
	WlanMacAddress              any         // wlan mac 地址
	DeviceIpAddress             any         // 设备 ip 地址
	ChargingStatus              any         // 设备充电状态
	BluetoothInfDevice          any         // 有关设备原始蓝牙信息
	BluetoothInfConnected       any         // 已连接蓝牙信息
	CameraTemperatureCelsius    any         // 相机的温度，摄氏度
	CameraTemperatureFahrenheit any         // 相机温度，华氏度
	LargespaceMapInfo           any         // 大空间地图信息
	Trackers                    any         // 追踪器
	Status                      any         // 1 上线 2 下线
	CreatedAt                   *gtime.Time //
	UpdatedAt                   *gtime.Time //
}
