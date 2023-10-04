import init, * as brotliWasm from "../../node_modules/brotli-dec-wasm/pkg/index.js";

import cache from "./dataCache";
import {derived, get, readable, Readable, writable, Writable} from "svelte/store"
import {getOS, getUsername, testBit} from "./utils";
import {Button, dpadKeyMap, keyMap} from "./consts";

const UncompressedFrameSize = 92160

export enum EventType {
    Frame,
    FramePatch,
    FrameSkip,
    ClientInfo,
    PatchCache,
    PatchCacheSync,
    FrameCache,
    FrameCacheSync,
    FrameSync,
    ClientListSync,
    ClientClosing,
    ClientListNew,
    ClientListIdentify,
    ServerInfo,
    PlayerInfo,
    PlayerIdentify,
}

export enum ControlEvent {
    Pause,
    PPU = 9,
    Info = 10,
    Close = 255,
}

export enum PPUControlEvent {
    ToggleBackground,
    ToggleWindow,
    ToggleSprites,
}

export enum InfoControlEvent {
    Compression = 1,
    CompressionLevel = 2,
    FramePatching = 3,
    FrameSkipping = 4,
    BackgroundToggle = 5,
    WindowToggle = 6,
    SpritesToggle = 7,
    FrameCaching = 10,
    RegisterUsername = 11,
    Player2Confirmation = 12
}

export enum PlayerInfoEvent {
    Paused,
    Status,
    BackgroundEnabled,
    WindowEnabled,
    SpritesEnabled,
}

type EventCallback = (event: EventType, data: Uint8Array) => void;

type OnConnect = () => void;
type OnDisconnect = () => void;
type OnMessage = (data: Uint8Array) => void;

/**
 * R
 */
class GameStats {
    transferRate: number;
    throughput: number;
}

class UserClient {
    clientIP: string;
    clientPort: number;
    userAgent: string;
    username: string;
    closing: boolean;
    id: number;

    os: string;
    ping: number;

    constructor(remoteAddr: string, userAgent: string, userName: string, id: number) {
        this.clientIP = remoteAddr.split(":")[0];
        this.clientPort = remoteAddr.split(":")[1] as number;
        this.username = userName;
        this.id = id
        this.ping = 0

        this.userAgent = userAgent;
        this.os = getOS(userAgent);
    }
}

export class Player {
    socket: Socket;
    patchCache: cache;
    frameCache: cache;
    paused: Writable<boolean>;
    isRunning: Writable<boolean>;
    hasSynced: boolean = false;

    framesPatched: Writable<number>;
    framesSkipped: Writable<number>;

    bgEnabled: Writable<boolean>;
    windowEnabled: Writable<boolean>;
    spritesEnabled: Writable<boolean>;

    patchSaved: Readable<number>;

    onEvent: EventCallback;
    onConnect: OnConnect;

    constructor(socket: Socket) {
        this.socket = socket;

        this.patchCache = new cache(16384);
        this.frameCache = new cache(1024);

        this.paused = writable(false);
        this.isRunning = writable(false);

        this.framesPatched = writable(0);
        this.framesSkipped = writable(0);

        this.patchSaved = derived([this.framesSkipped], ([$framesSkipped]) => {
            return $framesSkipped * UncompressedFrameSize
        })

        this.bgEnabled = writable(true);
        this.windowEnabled = writable(true);
        this.spritesEnabled = writable(true);
    }

    init(onEvent: EventCallback, onConnect: OnConnect) {
        this.onEvent = onEvent
        this.onConnect = onConnect

        this.bgEnabled.subscribe(bg => {
            this.send(ControlEvent.PPU, PPUControlEvent.ToggleBackground, new Uint8Array([bg ? 1 : 0]))
        })
        this.windowEnabled.subscribe(window => {
            this.send(ControlEvent.PPU, PPUControlEvent.ToggleWindow, new Uint8Array([window ? 1 : 0]))
        })
        this.spritesEnabled.subscribe(sprites => {
            this.send(ControlEvent.PPU, PPUControlEvent.ToggleSprites, new Uint8Array([sprites ? 1 : 0]))
        })
    }

    send(event: ControlEvent, subEvent : PPUControlEvent | InfoControlEvent = null, data: Uint8Array = new Uint8Array([])) : void {
        // don't send unless player has synced
        if (!this.hasSynced) return

        this.socket.socket.send(new Uint8Array([event, subEvent, ...data]))
    }

    togglePlayback() : void {
        this.paused.update(paused => {
            this.socket.socket.send(new Uint8Array([paused ? 1 : 0  ]))
            return !paused
        })
    }
}

const playerEvents: Array<EventType> = [
    EventType.Frame,
    EventType.FrameCache,
    EventType.FrameCacheSync,
    EventType.FrameSync,
    EventType.FramePatch,
    EventType.FrameSkip,
    EventType.PatchCache,
    EventType.PatchCacheSync,
    EventType.PlayerInfo,
]

class Game {
    socket: Socket;
    closing: Promise<null>;
    clientID: number;

    throughput: Writable<number>
    uncompressedThroughput: Writable<number>;
    transferRate: Readable<number>
    compression: Writable<boolean>;
    compressionLevel: Writable<number>;
    patchRatio: number;
    clients: Writable<Map<string,UserClient>>;
    client: Writable<UserClient>;
    username: Writable<string>;
    lastTransfer: number;
    speedData: Writable<Array<number>>

    player1: Player;
    player2: Player;
    isPlayer1: Writable<boolean>;
    isPlayer2: Writable<boolean>;

    frameCaching: Writable<boolean>;
    framePatching: Writable<boolean>;
    frameSkipping: Writable<boolean>;

    onEvent: EventCallback;
    onConnect: OnConnect;
    onDisconnect: OnDisconnect;

    constructor(url: string) {
        this.socket = new Socket(url);
        this.throughput = writable(0);
        this.uncompressedThroughput = writable(0);
        this.compression = writable(true);
        this.compressionLevel = writable(1);
        this.clients = writable(new Map<string,UserClient>())
        this.username = writable("");
        this.client = writable(new UserClient("", "", "", 0))

        this.frameCaching = writable(true);
        this.framePatching = writable(true);
        this.frameSkipping = writable(true);

        this.speedData = writable(new Array<number>());

        this.lastTransfer = 0;

        this.transferRate = readable(0, set => {
            let interval = setInterval(() => {
                let throughput;
                this.uncompressedThroughput.subscribe(t => throughput = t)();
                set((throughput - this.lastTransfer))

                this.speedData.update(data => {
                    data.push(throughput - this.lastTransfer)

                    // make sure it doesn't exceed 60 items in length (1 min)
                    if (data.length > 60) {
                        data.shift()
                    }

                    return data
                })

                this.lastTransfer = throughput
            }, 1000)

            return () => {
                clearInterval(interval)
            }
        })

        this.player1 = new Player(this.socket);
        this.player2 = new Player(this.socket);
        this.isPlayer1 = writable(false);
        this.isPlayer2 = writable(false);
        // this.username = getUsername();
    }

    async connect(): Promise<any> {
        await this.socket.init(data => {
            this.uncompressedThroughput.update(t => t+= data.byteLength)
            // TODO handle cache sync

            // get event type and data
            let eventType = data[0];
            let eventData = data.slice(1);

            let player: Player;

            if (playerEvents.some(e => e == eventType)) {
                // player id should be the second byte
                if (eventData[0] == 1) {
                    player = this.player1
                } else if (eventData[0] == 2) {
                    player = this.player2
                } else {
                    console.warn("processing player event but no player id found", eventType, eventData[0], eventData)
                }

                // remove player id from data
                eventData = eventData.slice(1)
            }

            // frame and patch is prefixed with 2 byte index of cache entry
            let cacheView: DataView;
            if (eventType === EventType.Frame || eventType === EventType.FramePatch) {
                let cacheIdx = eventData.slice(0, 2)
                eventData = eventData.slice(2)
                cacheView = new DataView(cacheIdx.buffer)
            }

            switch (eventType) {
                case EventType.ClientInfo:
                    // special case to handle initial compression
                    switch(eventData[0]) {
                        case 1:
                            this.compression.set(eventData[1] === 1)
                            break
                        case 2:
                            this.compressionLevel.set(eventData[1])
                            break
                        case 3:
                            this.framePatching.set(eventData[1] === 1)
                            break
                        case 4:
                            this.frameSkipping.set(eventData[1] === 1)
                            break
                        case 5:
                            this.player1.isRunning.set(testBit(eventData[1], 1))
                            this.player2.isRunning.set(testBit(eventData[1], 2))
                            this.compression.set(testBit(eventData[1], 3))
                            this.framePatching.set(testBit(eventData[1], 4))
                            this.frameSkipping.set(testBit(eventData[1], 5))
                            this.frameCaching.set(testBit(eventData[1], 7))

                            this.compressionLevel.set(eventData[2])
                            this.patchRatio = eventData[3]

                            break
                        case 10:
                            this.frameCaching.set(eventData[1] === 1)
                            break
                        case InfoControlEvent.Player2Confirmation:
                            break
                        case InfoControlEvent.RegisterUsername:
                            let text = new TextDecoder().decode(eventData.slice(1))
                            if (eventData[1] === 255) {
                                this.client.update(client => {
                                    let [remoteAddr, userAgent, userName, id] = text.slice(1).split("\x00")
                                    client.clientIP = remoteAddr.split(":")[0]
                                    client.clientPort = remoteAddr.split(":")[1] as number
                                    client.userAgent = userAgent
                                    client.os = getOS(userAgent)
                                    client.username = userName
                                    client.id = new DataView(new TextEncoder().encode(id).buffer).getUint8(0)

                                    this.clientID = client.id
                                    return client;
                                })
                            } else {
                                let [remoteAddr, userAgent, userName, id] = text.split(`\x00`)
                                this.clients.update(clients => {
                                    clients.set(remoteAddr, new UserClient(remoteAddr, userAgent, userName, new DataView(new TextEncoder().encode(id).buffer).getUint8(0)))

                                    return clients;
                                })
                            }
                            break
                    }
                    break
                case EventType.PlayerInfo:
                    switch (eventData[0]) {
                        case PlayerInfoEvent.Paused:
                            player.paused.set(eventData[1] === 0)
                            break
                        case PlayerInfoEvent.Status:
                            // TODO implement this
                            /* infoPlayer.bgEnabled.set(!testBit(eventData[2], 0))
                            infoPlayer.windowEnabled.set(!testBit(eventData[2], 1))
                            infoPlayer.spritesEnabled.set(!testBit(eventData[2], 2))
                            infoPlayer.paused.set(!testBit(eventData[2], 6))
                            infoPlayer.hasSynced = true
                             */
                            break
                        case PlayerInfoEvent.BackgroundEnabled:
                            player.bgEnabled.set(eventData[1] === 1)
                            break
                        case PlayerInfoEvent.WindowEnabled:
                            player.windowEnabled.set(eventData[1] === 1)
                            break
                        case PlayerInfoEvent.SpritesEnabled:
                            player.spritesEnabled.set(eventData[1] === 1)
                            break
                    }
                    break
                case EventType.Frame:
                    // cache frame
                    player.frameCache.put(cacheView.getUint16(0, true), eventData)
                    break
                case EventType.FrameSync:
                    // frame sync is essentially the same as a Frame event, but without idx, so just change event here
                    eventType = EventType.Frame
                    break
                case EventType.FramePatch:
                    // cache patch
                    player.patchCache.put(cacheView.getUint16(0, true), eventData)
                    player.framesPatched.update(patched => patched++)
                    break
                case EventType.FrameCache:
                case EventType.PatchCache:
                    // which event are we processing? to access the proper cache
                    if (eventType === EventType.FrameCache) {
                        eventData = player.frameCache.get(padToUint16(eventData))
                        eventType = EventType.Frame
                    } else if (eventType === EventType.PatchCache) {
                        eventData = player.patchCache.get(padToUint16(eventData))
                        eventType = EventType.FramePatch
                        player.framesPatched.update(patched => patched+=1)
                    }

                    // did we manage to successfully get item from cache?
                    if (eventData === null || eventData.length === 0) {
                        console.warn(`failed to retrieve item from cache ${eventType}: ${cacheView.getUint16(0, true)}`)
                        return
                    }
                    break
                case EventType.FrameCacheSync:
                case EventType.PatchCacheSync:
                    let now = new Date()
                    // determine which cache we are operating on
                    let workingCache: cache;
                    if (eventType === EventType.FrameCacheSync) {
                        workingCache = player.frameCache
                    } else if (eventType === EventType.PatchCacheSync) {
                        workingCache = player.patchCache
                    }

                    let cached = 0;
                    let tempBuffer: Array<Uint8Array> = new Array(workingCache.capacity)

                    // iterate over data
                    let dataIdx: number = 0;
                    let view = new DataView(eventData.buffer)

                    while (dataIdx < eventData.length) {
                        let length = view.getUint16(dataIdx, true)
                        let idx = view.getUint16(dataIdx+2, true)
                        let frameData = eventData.slice(dataIdx+4, dataIdx+length+4)

                        // move data pointer
                        dataIdx += length + 4

                        if (frameData.length === 0) {
                            console.warn("FrameCacheSync request for empty frame", length, idx, frameData)
                            break
                        }

                        tempBuffer[idx] = frameData

                        // break if no more data
                        if (dataIdx === eventData.length) {
                            break
                        }
                    }

                    workingCache.batchPut(tempBuffer)

                    break
                case EventType.FrameSkip:
                    player.framesSkipped.update(skipped => skipped += padToUint32(eventData))

                    break
                case EventType.ClientListIdentify:
                case EventType.ClientListSync:
                case EventType.ClientListNew:
                    let [...results] = new TextDecoder().decode(eventData).split(`\n`);
                    results.forEach(result => {
                        let [remoteAddr, userAgent, userName, id] = result.split(`\x00`)
                        console.log(new TextEncoder().encode(result))
                        this.clients.update(clients => {
                            clients.set(remoteAddr, new UserClient(remoteAddr, userAgent, userName, new DataView(new TextEncoder().encode(id).buffer).getUint8(0)))

                            return clients;
                        })
                    })

                    break
                case EventType.ClientClosing:
                    const clientIdentifier = new TextDecoder().decode(eventData)
                    console.log("closing", clientIdentifier)
                    this.clients.update(clients => {
                        clients.forEach(client => {
                            if (`${client.clientIP}:${client.clientPort}` === clientIdentifier) {
                                client.closing = true
                            }
                        })

                        return clients;
                    })
                    setTimeout(() => {
                        this.clients.update(clients => {
                            clients.delete(clientIdentifier)

                            return clients;
                        })
                    }, 2500)

                    break
                case EventType.ServerInfo:
                    let infoIdx: number = 0;
                    let dataView = new DataView(eventData.buffer)
                    while (infoIdx < eventData.byteLength) {
                        // info consists of 1 byte for ID, 2 bytes for uint16 ping
                        let id = eventData[infoIdx]
                        let ping = dataView.getUint16(infoIdx+1, true)

                        if (this.clientID === id) {
                            this.client.update(c => {
                                c.ping = ping

                                return c
                            })
                        } else {
                            this.clients.update(cs => {
                                cs.forEach(client => {
                                    if (client.id === id) {
                                        client.ping = ping
                                        return
                                    }
                                })

                                return cs
                            })
                        }

                        infoIdx += 3
                    }
                    break
                case EventType.PlayerIdentify:
                    console.log("client becomes player")
                    if (eventData[0] == 1) {
                        this.isPlayer1.set(true)
                    } else if (eventData[0] == 2) {
                        this.isPlayer2.set(true)
                    } else {
                        console.warn("unhandled player type", eventData[0])
                    }
                    break
                default:
                    console.warn("unhandled event", eventType)
                    break
            }

            // handle compression if necessary
            if (eventType === EventType.Frame || eventType === EventType.FramePatch || eventType === EventType.FrameSync ) {
                if (this.compression) {
                    try {
                        eventData = brotliWasm.decompress(eventData)
                    } catch (e: Error) {
                        console.warn(`failed to decompress ${eventType}`)
                        return
                    }
                }
            }

            // handle throughput
            this.throughput.update(throughput => throughput + eventData.length)

            if (eventType === EventType.Frame || eventType === EventType.FramePatch) {
                // event callback
                player.onEvent(eventType, eventData);
            }
        }, this.onConnect, this.onDisconnect);
    }

    async init(onEvent: EventCallback, onConnect: OnConnect, onDisconnect: OnDisconnect): Promise<any> {
        this.username.set(getUsername())

        this.onEvent = onEvent;
        this.onConnect = onConnect
        this.onDisconnect = onDisconnect;

        await init(new URL('/index_bg.wasm', document.location.href)).then(() => brotliWasm)
        await this.connect();

        let username: string;
        this.username.subscribe(name => {
            username = name
        })()
        this.socket.socket.send(new Uint8Array([ControlEvent.Info, InfoControlEvent.RegisterUsername, ...new TextEncoder().encode(username)]))

        this.frameCaching.subscribe(cache => {
            this.send(ControlEvent.Info, InfoControlEvent.FrameCaching, new Uint8Array([cache ? 1 : 0]))
        })
        this.framePatching.subscribe(patch => {
            this.send(ControlEvent.Info, InfoControlEvent.FramePatching, new Uint8Array([patch ? 1 : 0]))
        })
        this.frameSkipping.subscribe(skip => {
            this.send(ControlEvent.Info, InfoControlEvent.FrameSkipping, new Uint8Array([skip ? 1 : 0]))
        })

        this.compressionLevel.subscribe(level => {
            this.send(ControlEvent.Info, InfoControlEvent.CompressionLevel, new Uint8Array([level]))
        })

        window.addEventListener("beforeunload", e => {
            this.close()
        })
        document.addEventListener("keydown", event => {
            if (event.key in keyMap) {
                this.socket.socket.send(new Uint8Array([keyMap[event.key], 1]))

                document.querySelector(`.${dpadKeyMap[event.key]}`)?.classList.add("on")
            }
        })
        document.addEventListener("keyup", event => {
            if (event.key in keyMap) {
                this.socket.socket.send(new Uint8Array([keyMap[event.key], 0]))

                document.querySelector(`.${dpadKeyMap[event.key]}`)?.classList.remove("on")
            }
        })
    }

    send(event: ControlEvent, subEvent: PPUControlEvent | InfoControlEvent = null, data: Uint8Array = new Uint8Array([])) : void {
        this.socket.socket.send(new Uint8Array([event, subEvent, ...data]))
    }

    press(button: Button) : void {
        this.socket.socket.send(new Uint8Array([button, 1]))
    }

    release(button: Button) : void {
        this.socket.socket.send(new Uint8Array([button, 0]))
    }

    close() {
        this.send(ControlEvent.Close)
        this.socket.socket.close(1000);
    }
}

class Socket {
    url: string;
    socket: WebSocket;
    connected: Writable<boolean>;
    transferred: Writable<number>;

    onConnect: OnConnect;
    onDisconnect: OnDisconnect;
    onMessage: OnMessage;

    constructor(url: string) {
        this.url = url;
        this.connected = writable(false);
        this.transferred = writable(0);
    }

    async init(onMessage: OnMessage = null, onConnect: OnConnect = null, onDisconnect: OnDisconnect = null): Promise<null> {
        return new Promise((resolve, reject) => {
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
                this.connected.set(true);
                if (this.onConnect !== null) {
                    this.onConnect();
                }

                resolve();
            });
            this.socket.addEventListener("close", () => {
                this.connected.set(false);
                if (this.onDisconnect !== undefined) {
                    this.onDisconnect();
                }
            });
            this.socket.addEventListener("message", async (event) => {
                let eventData = await event.data.arrayBuffer()
                if (this.onMessage !== null) {
                    this.onMessage(new Uint8Array(eventData));
                }
                this.transferred.update(transferred => {
                    return transferred + eventData.byteLength;
                });
            });
        })
    }

}

function padToUint16(b: Uint8Array) {
    let buffer = new ArrayBuffer(2);
    let view = new Uint8Array(buffer);
    for (let i = 0; i < b.length; i++) {
        view[i] = b[i]
    }

    return new Uint16Array(buffer)[0]
}

function padToUint32(b: Uint8Array) {
    let buffer = new ArrayBuffer(4);
    let view = new Uint8Array(buffer);
    for (let i = 0; i < b.length; i++) {
        view[i] = b[i]
    }

    return new Uint32Array(buffer)[0]
}

export default new Game("ws://192.168.1.154:8090/")

export let adminView = writable(true);