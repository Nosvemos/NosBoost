package config

// DeviceBackupState holds Message Signaled Interrupt settings for a specific hardware device.
type DeviceBackupState struct {
	DevicePath           string `json:"device_path"`
	MSISupportedExists   bool   `json:"msi_supported_exists"`
	MSISupportedValue    uint32 `json:"msi_supported_value"`
	DevicePriorityExists bool   `json:"device_priority_exists"`
	DevicePriorityValue  uint32 `json:"device_priority_value"`
}

// NICBackupState holds low-latency TCP configuration overrides for a specific network adapter.
type NICBackupState struct {
	InterfaceGUID         string `json:"interface_guid"`
	TcpAckFrequencyExists bool   `json:"tcp_ack_frequency_exists"`
	TcpAckFrequencyValue  uint32 `json:"tcp_ack_frequency_value"`
	TCPNoDelayExists      bool   `json:"tcp_nodelay_exists"`
	TCPNoDelayValue       uint32 `json:"tcp_nodelay_value"`
	TcpDelAckTicksExists  bool   `json:"tcp_delack_ticks_exists"`
	TcpDelAckTicksValue   uint32 `json:"tcp_delack_ticks_value"`
}

// NetworkBackupState tracks system-wide and adapter-specific latency optimizations.
type NetworkBackupState struct {
	NICs                       []NICBackupState `json:"nics"`
	NetworkThrottlingExists    bool             `json:"network_throttling_exists"`
	NetworkThrottlingValue     uint32           `json:"network_throttling_value"`
	SystemResponsivenessExists bool             `json:"system_responsiveness_exists"`
	SystemResponsivenessValue  uint32           `json:"system_responsiveness_value"`
}

// PowerBackupState holds the original active power plan configuration.
type PowerBackupState struct {
	OriginalActiveScheme string `json:"original_active_scheme"`
	MinCoresACExists     bool   `json:"min_cores_ac_exists"`
	MinCoresACValue      uint32 `json:"min_cores_ac_value"`
	MinCoresDCExists     bool   `json:"min_cores_dc_exists"`
	MinCoresDCValue      uint32 `json:"min_cores_dc_value"`
	MaxCoresACExists     bool   `json:"max_cores_ac_exists"`
	MaxCoresACValue      uint32 `json:"max_cores_ac_value"`
	MaxCoresDCExists     bool   `json:"max_cores_dc_exists"`
	MaxCoresDCValue      uint32 `json:"max_cores_dc_value"`
}

// ServiceBackupState represents the service state and startup parameters for target background services.
type ServiceBackupState struct {
	ServiceName string `json:"service_name"`
	StartExists bool   `json:"start_exists"`
	StartValue  uint32 `json:"start_value"`
}

// GamesTaskBackupState stores original multimedia scheduling parameters for games.
type GamesTaskBackupState struct {
	AffinityExists           bool   `json:"affinity_exists"`
	AffinityValue            uint32 `json:"affinity_value"`
	BackgroundOnlyExists     bool   `json:"background_only_exists"`
	BackgroundOnlyValue      string `json:"background_only_value"`
	ClockRateExists          bool   `json:"clock_rate_exists"`
	ClockRateValue           uint32 `json:"clock_rate_value"`
	GPUPriorityExists        bool   `json:"gpu_priority_exists"`
	GPUPriorityValue         uint32 `json:"gpu_priority_value"`
	PriorityExists           bool   `json:"priority_exists"`
	PriorityValue            uint32 `json:"priority_value"`
	SchedulingCategoryExists bool   `json:"scheduling_category_exists"`
	SchedulingCategoryValue  string `json:"scheduling_category_value"`
	SFIOPriorityExists       bool   `json:"sfio_priority_exists"`
	SFIOPriorityValue        string `json:"sfio_priority_value"`
}

// SystemBaselineState is the primary container for a full restoration point of the Windows OS.
type SystemBaselineState struct {
	Version                      string               `json:"version"`
	Timestamp                    string               `json:"timestamp"`
	Devices                      []DeviceBackupState  `json:"devices"`
	Network                      NetworkBackupState   `json:"network"`
	Power                        PowerBackupState     `json:"power"`
	Services                     []ServiceBackupState `json:"services"`
	GamesTask                    GamesTaskBackupState `json:"games_task"`
	Win32PrioritySeparationExist bool                 `json:"win32_priority_separation_exist"`
	Win32PrioritySeparationValue uint32               `json:"win32_priority_separation_value"`
	MouseQueueExist              bool                 `json:"mouse_queue_exist"`
	MouseQueueValue              uint32               `json:"mouse_queue_value"`
	KeyboardQueueExist           bool                 `json:"keyboard_queue_exist"`
	KeyboardQueueValue           uint32               `json:"keyboard_queue_value"`
	MouseSpeedExists             bool                 `json:"mouse_speed_exists"`
	MouseSpeedValue              string               `json:"mouse_speed_value"`
	MouseThreshold1Exists        bool                 `json:"mouse_threshold1_exists"`
	MouseThreshold1Value         string               `json:"mouse_threshold1_value"`
	MouseThreshold2Exists        bool                 `json:"mouse_threshold2_exists"`
	MouseThreshold2Value         string               `json:"mouse_threshold2_value"`
	KeyboardDelayExists          bool                 `json:"keyboard_delay_exists"`
	KeyboardDelayValue           string               `json:"keyboard_delay_value"`
	KeyboardSpeedExists          bool                 `json:"keyboard_speed_exists"`
	KeyboardSpeedValue           string               `json:"keyboard_speed_value"`
	GameDVREnabledExists         bool                 `json:"gamedvr_enabled_exists"`
	GameDVREnabledValue          uint32               `json:"gamedvr_enabled_value"`
	AppCaptureEnabledExists      bool                 `json:"app_capture_enabled_exists"`
	AppCaptureEnabledValue       uint32               `json:"app_capture_enabled_value"`
	MaxUserPortExists            bool                 `json:"max_user_port_exists"`
	MaxUserPortValue             uint32               `json:"max_user_port_value"`
	TcpNumConnectionsExists      bool                 `json:"tcp_num_connections_exists"`
	TcpNumConnectionsValue       uint32               `json:"tcp_num_connections_value"`
	Tcp1323OptsExists            bool                 `json:"tcp_1323_opts_exists"`
	Tcp1323OptsValue             uint32               `json:"tcp_1323_opts_value"`
	DisablePagingExecutiveExists bool                 `json:"disable_paging_executive_exists"`
	DisablePagingExecutiveValue  uint32               `json:"disable_paging_executive_value"`
	LargeSystemCacheExists       bool                 `json:"large_system_cache_exists"`
	LargeSystemCacheValue        uint32               `json:"large_system_cache_value"`
}
const BackupFileName = "state_backup.json"
