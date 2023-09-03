import init, * as brotliWasm from "../../node_modules/brotli-dec-wasm/pkg/index.js";
import cache from "$lib/dataCache.js";


import XXH64 from "xxhashjs";
//import { decompress } from "@bokuweb/zstd-wasm";


class Socket {
    url = '';
    socket = null;
    connected = false;
    compression = true;
    patchCache = new cache();
    frameCache = new cache();
    isSynced = false;
    onConnect = function(){};
    onDisconnect = function(){};
    onMsg = function(event, data, msg){};

    rawTransfer = 0
    uncompressedTransfer = 0

    constructor(url = null, onMsg = null) {
        this.url = url
        this.patchCache = new cache(16384);
        this.frameCache = new cache(1024)

        if (onMsg) {
            this.onMsg = onMsg;
        }
    }

    async init(onMsg = null, onConnect = null, onDisconnect = null) {
        await init(new URL('http://localhost:5173/index_bg.wasm')).then(() => brotliWasm)

        if (onMsg != null) {
            this.onMsg = onMsg
        }
        if (onConnect != null) {
            this.onConnect = onConnect;
        }
        if (onDisconnect != null) {
            this.onDisconnect = onDisconnect;
        }

        this.socket = new WebSocket(this.url);
        this.socket.addEventListener('open', () => {
            this.connected = true;
            this.onConnect();
        });
        this.socket.addEventListener('close', () => {
            this.connected = false;
            this.onDisconnect();


        });
        this.socket.addEventListener('message', async event => {
            let buf = await event.data.arrayBuffer()
            let tArray = new Uint8Array(buf)

            // get the message type
            let type = tArray[0]

            if (!this.isSynced && (type === FramePatch)) {
                console.warn("received message before sync")
                return
            }

            // get the message data
            let data = tArray.slice(1)

            let cacheView;
            // if type is FramePatch, we need to add to cache
            if (type === Frame) {
                // first 2 bytes of frame is the cache index
                let cacheIdx = data.slice(0, 2)
                data = data.slice(2)
                cacheView = new DataView(cacheIdx.buffer)
            }
            if (type === FramePatch) {
                // first 2 bytes of framepatch is the cache index
                let cacheIdx = data.slice(0, 2)
                data = data.slice(2)
                cacheView = new DataView(cacheIdx.buffer)
            }



            switch (type) {
                case Info:
                    switch (data[0]) {
                        case 1:
                            // compression
                            this.compression = data[1] === 1
                            break
                    }
                    break
                case FrameSync:
                    type = Frame
                    break
                case Frame:
                    this.frameCache.put(cacheView.getUint16(0, true), data)
                    break;
                case FramePatch:
                    this.patchCache.put(cacheView.getUint16(0, true), data)
                    break;
                case FrameCache:
                case PatchCache:
                    let buffer = new ArrayBuffer(2);
                    let view = new Uint8Array(buffer)
                    let originalView = new Uint8Array(data.buffer)
                    for (let i = 0; i < 2; i++) {
                        view[i] = originalView[i]
                    }

                    if (type === FrameCache) {
                        data = this.frameCache.get(new DataView(buffer).getUint16(0, true))
                    } else {
                        data = this.patchCache.get(new DataView(buffer).getUint16(0, true))
                    }
                    if (data === null || data.length === 0) {
                        console.warn("FrameCache request for empty frame", new DataView(buffer).getUint16(0, true), data)
                        // console.log(this.cache)
                        return;
                    }
                    type = FramePatch
                    break
                case PatchCacheSync:
                    for (let i = 0; i < this.patchCache.capacity; i++) {
                        const length = new DataView(data.buffer).getUint16(0, true)
                        const idx = new DataView(data.buffer.slice(2)).getUint16(0, true)
                        const frameData = data.slice(4, length+4)

                        // move data pointer
                        data = data.slice(length + 4)

                        if (frameData.length === 0) {
                            console.warn("PatchCacheSync request for empty frame", i, length, idx, frameData)
                            break
                        }

                        // add data to cache
                        this.patchCache.put(idx, frameData)

                        // break if no more data
                        if (data.length === 0) {
                            break
                        }
                    }
                    this.isSynced = true
                    return
                case FrameCacheSync:
                    for (let i = 0; i < this.frameCache.capacity; i++) {
                        const length = new DataView(data.buffer).getUint16(0, true)
                        const idx = new DataView(data.buffer.slice(2)).getUint16(0, true)
                        const frameData = data.slice(4, length+4)

                        // move data pointer
                        data = data.slice(length + 4)

                        if (frameData.length === 0) {
                            console.warn("FrameCacheSync request for empty frame", i, length, idx, frameData)
                            break
                        }

                        // add data to cache
                        this.frameCache.put(idx, frameData)

                        // break if no more data
                        if (data.length === 0) {
                            break
                        }
                    }
                    return
                case FrameSkip:
                    break
            }

            // if type is Frame or FramePatch, we need to decompress
            if (type === Frame || type === FramePatch || type == FrameSync) {
                if (this.compression) {
                    try {
                        data = brotliWasm.decompress(data)
                    } catch (e) {
                        if (String(e).includes("Src size is incorrect")) {
                            window.location.reload() // TODO: handle this better
                        } else {
                            console.warn(e)
                        }
                    }
                }
            }

            // data accounting
            this.rawTransfer += buf.byteLength
            this.uncompressedTransfer += data.length


            this.onMsg(type, data, event)
        })

    }

    send(data) {
        if (this.socket) {
            this.socket.send(data);
        }
    }

    get connected() {
        return this.connected
    }

    set connected(value) {
        this.connected = value
    }
}

export default Socket;

export const Frame = 0
export const FramePatch = 1
export const FrameSkip = 2
export const Info = 3
export const PatchCache = 4
export const PatchCacheSync = 5
export const FrameCache = 6
export const FrameCacheSync = 7
export const FrameSync = 8