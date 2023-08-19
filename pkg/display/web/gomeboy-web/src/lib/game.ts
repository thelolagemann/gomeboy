import cache from "../socket/cache";

enum EventType {
    Frame,
    FramePatch,
    FrameSkip,
    Info,
    PatchCache,
    PatchCacheSync,
    FrameCache,
    FrameCacheSync,
    FrameSync,
}

type EventCallback = (event: EventType, data: Uint8Array) => void;

type OnConnect = () => void;
type OnDisconnect = () => void;
type OnMessage = (data: Uint8Array) => void;

class Game {
    socket: Socket;
    patchCache: cache;
    frameCache: cache;
    onEvent: EventCallback;

    constructor(url: string, onEvent: EventCallback) {
        this.socket = new Socket(url);
        this.patchCache = new cache(16384);
        this.frameCache = new cache(2048);

        this.onEvent = onEvent;

        this.socket.socket.addEventListener("message", (event) => {
            // first byte of data is the event type, remaining is any data
            let data = new Uint8Array(event.data);
            const eventType = data[0];
            data = data.slice(1)
            this.onEvent(eventType, data);
        })
    }

}

class Socket {
    url: string;
    socket: WebSocket;
    connected: boolean;

    onConnect: OnConnect;
    onDisconnect: OnDisconnect;
    onMessage: OnMessage;

    constructor(url: string) {
        this.url = url;
        this.connected = false;
    }

    async init(onMessage: OnMessage = null, onConnect: OnConnect = null, onDisconnect: OnDisconnect = null): Promise<any> {
        if (onMessage !== null) {
            this.onMessage = onMessage
        }
        if (onConnect !== null) {
            this.onConnect = onConnect;
        }
        if (onDisconnect !== null) {
            this.onDisconnect = onDisconnect;
        }

        this.socket = new WebSocket(this.url);
        this.socket.addEventListener("open", () => {
            this.connected = true;
            if (this.onConnect !== null) {
                this.onConnect();
            }
        });
        this.socket.addEventListener("close", () => {
            this.connected = false;
            if (this.onDisconnect !== null) {
                this.onDisconnect();
            }
        });
        this.socket.addEventListener("message", (event) => {
            if (this.onMessage !== null) {
                this.onMessage(event.data);
            }
        });
    }
}