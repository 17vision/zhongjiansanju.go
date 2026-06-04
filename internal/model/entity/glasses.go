// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// Glasses is the golang structure for table glasses.
type Glasses struct {
	Id                          uint64      `json:"id"                          orm:"id"                            description:""`            //
	Name                        string      `json:"name"                        orm:"name"                          description:"设备名称"`        // 设备名称
	Code                        string      `json:"code"                        orm:"code"                          description:"设备编号"`        // 设备编号
	BatteryLevel                string      `json:"batteryLevel"                orm:"battery_level"                 description:"电量"`          // 电量
	SystemVersion               string      `json:"systemVersion"               orm:"system_version"                description:"系统版本"`        // 系统版本
	EquipmentModel              string      `json:"equipmentModel"              orm:"equipment_model"               description:"设备型号"`        // 设备型号
	EquipmentSn                 string      `json:"equipmentSn"                 orm:"equipment_sn"                  description:"设备序列号"`       // 设备序列号
	CustomerSn                  string      `json:"customerSn"                  orm:"customer_sn"                   description:"客户序列号"`       // 客户序列号
	InternalStorageSpace        string      `json:"internalStorageSpace"        orm:"internal_storage_space"        description:"设备存储"`        // 设备存储
	BluetoothStatus             string      `json:"bluetoothStatus"             orm:"bluetooth_status"              description:"蓝牙状态"`        // 蓝牙状态
	BluetoothName               string      `json:"bluetoothName"               orm:"bluetooth_name"                description:"蓝牙名称"`        // 蓝牙名称
	BluetoothMacAddress         string      `json:"bluetoothMacAddress"         orm:"bluetooth_mac_address"         description:"蓝牙mac地址"`     // 蓝牙mac地址
	WifiStatus                  string      `json:"wifiStatus"                  orm:"wifi_status"                   description:"wifi状态"`      // wifi状态
	WifiNameConnected           string      `json:"wifiNameConnected"           orm:"wifi_name_connected"           description:"已连接的wifi状态"`  // 已连接的wifi状态
	WlanMacAddress              string      `json:"wlanMacAddress"              orm:"wlan_mac_address"              description:"wlan mac 地址"` // wlan mac 地址
	DeviceIpAddress             string      `json:"deviceIpAddress"             orm:"device_ip_address"             description:"设备 ip 地址"`    // 设备 ip 地址
	ChargingStatus              string      `json:"chargingStatus"              orm:"charging_status"               description:"设备充电状态"`      // 设备充电状态
	BluetoothInfDevice          string      `json:"bluetoothInfDevice"          orm:"bluetooth_inf_device"          description:"有关设备原始蓝牙信息"`  // 有关设备原始蓝牙信息
	BluetoothInfConnected       string      `json:"bluetoothInfConnected"       orm:"bluetooth_inf_connected"       description:"已连接蓝牙信息"`     // 已连接蓝牙信息
	CameraTemperatureCelsius    float64     `json:"cameraTemperatureCelsius"    orm:"camera_temperature_celsius"    description:"相机的温度，摄氏度"`   // 相机的温度，摄氏度
	CameraTemperatureFahrenheit float64     `json:"cameraTemperatureFahrenheit" orm:"camera_temperature_fahrenheit" description:"相机温度，华氏度"`    // 相机温度，华氏度
	LargespaceMapInfo           string      `json:"largespaceMapInfo"           orm:"largespace_map_info"           description:"大空间地图信息"`     // 大空间地图信息
	Trackers                    string      `json:"trackers"                    orm:"trackers"                      description:"追踪器"`         // 追踪器
	Status                      uint        `json:"status"                      orm:"status"                        description:"1 上线 2 下线"`   // 1 上线 2 下线
	UseStatus                   uint        `json:"useStatus"                   orm:"use_status"                    description:"1 可用 2 不可用"`  // 1 可用 2 不可用
	CreatedAt                   *gtime.Time `json:"createdAt"                   orm:"created_at"                    description:""`            //
	UpdatedAt                   *gtime.Time `json:"updatedAt"                   orm:"updated_at"                    description:""`            //
}
