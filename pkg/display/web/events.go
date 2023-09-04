package web

type Event = uint8

const (
	_ Event = iota
	Compression
	CompressionLevel
	FramePatching
	FrameSkipping
	ClientStatus
	BackgroundDisabled
	WindowDisabled
	SpritesDisabled
	FramePatchingRatio
	FrameCaching
	RegisterUsername
	Player2Confirmation
	KeepAlive = 254
	Closing   = 255
)

type PlayerEvent = uint8

const (
	PausePlay PlayerEvent = iota
	Status
	BackgroundEnabled
	WindowEnabled
	SpritesEnabled
)

type Type = uint8

const (
	Frame Type = iota
	FramePatch
	FrameSkip
	ClientInfo
	PatchCache
	PatchCacheSync
	FrameCache
	FrameCacheSync
	FrameSync
	ClientListSync
	ClientClosing
	ClientListNew
	ClientListIdentify
	ServerInfo
	PlayerInfo
	PlayerIdentify
)
