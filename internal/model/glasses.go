package model

import "github.com/gogf/gf/v2/os/gtime"

type Glasses struct {
	Name                        string  `json:"name" v:"length:1,100#设备名称不能为空|设备名称长度不能超过100字符" dc:"设备名称"`
	EquipmentSn                 string  `json:"equipmentSn" v:"required|length:1,50#设备序列号不能为空|序列号长度不能超过50字符" dc:"设备唯一序列号"`
	BatteryLevel                string  `json:"batteryLevel" dc:"当前电量，如：85%"`
	SystemVersion               string  `json:"systemVersion" dc:"设备系统版本"`
	EquipmentModel              string  `json:"equipmentModel" dc:"设备型号"`
	CustomerSn                  string  `json:"customerSn" dc:"客户自定义序列号"`
	InternalStorageSpace        string  `json:"internalStorageSpace" dc:"设备存储容量，如：128GB"`
	BluetoothStatus             string  `json:"bluetoothStatus" dc:"蓝牙状态：已连接/未连接"`
	BluetoothName               string  `json:"bluetoothName" dc:"蓝牙设备名称"`
	BluetoothMacAddress         string  `json:"bluetoothMacAddress" dc:"蓝牙MAC地址"`
	WifiStatus                  string  `json:"wifiStatus" dc:"WiFi状态：已连接/未连接"`
	WifiNameConnected           string  `json:"wifiNameConnected" dc:"已连接的WiFi名称"`
	WlanMacAddress              string  `json:"wlanMacAddress" dc:"无线网卡MAC地址"`
	DeviceIpAddress             string  `json:"deviceIpAddress" dc:"设备局域网IP地址"`
	ChargingStatus              string  `json:"chargingStatus" dc:"充电状态：充电中/未充电"`
	BluetoothInfDevice          string  `json:"bluetoothInfDevice" dc:"原始蓝牙设备信息（JSON字符串）"`
	BluetoothInfConnected       string  `json:"bluetoothInfConnected" dc:"已连接蓝牙设备信息（JSON字符串）"`
	CameraTemperatureCelsius    float64 `json:"cameraTemperatureCelsius" dc:"相机温度（摄氏度）"`
	CameraTemperatureFahrenheit float64 `json:"cameraTemperatureFahrenheit" dc:"相机温度（华氏度）"`
	LargespaceMapInfo           string  `json:"largespaceMapInfo" dc:"大空间地图数据（JSON字符串）"`
	Trackers                    string  `json:"trackers" dc:"追踪器信息（JSON字符串）"`
	Status                      uint    `json:"status" v:"in:1,2#状态只能是1(上线)或2(下线)" dc:"设备状态：1-上线，2-下线（默认2）"`
}

type GlassesCreateReq struct {
	Glasses
}

type GlassesCreateRes struct {
	Id        uint64      `json:"id" dc:"设备ID"`
	CreatedAt *gtime.Time `json:"createdAt" dc:"创建时间"`
}

type GlasseUpdateReq struct {
	Id uint64 `json:"id" v:"required#设备ID不能为空" dc:"设备ID"`
	Glasses
}
