import {derived, readable, Readable, writable, Writable} from "svelte/store";

class DataCache {
    capacity: number;
    hits: Writable<number>;
    map: Writable<Map<number, Uint8Array>>;
    averageSize: Readable<number>;
    size: Readable<number>;
    length: Readable<number>;

    constructor(capacity: number) {
        this.capacity = capacity;
        this.hits = writable(0);
        this.map = writable(new Map());

        this.size = readable(0, set => {
            let interval = setInterval(() => {
                this.map.subscribe(map => {
                    let size = 0;
                    for (const [_, value] of map) {
                        size += value.byteLength;
                    }
                    set(size);
                })()
            }, 1000)

            return () => {
                clearInterval(interval);
            }
        })
        this.averageSize = derived([this.size, this.map], ([$size, $map]) => {
            return Math.trunc($size / $map.size);
        })
        this.length = derived([this.map], ([$map]) => {
            return $map.size;
        })
    }

    get(key: number) : Uint8Array | null {
        let result: Uint8Array = null;
        this.map.subscribe(map => {
            if (map.has(key)) {
                this.hits.update(hits => hits + 1);
                result = map.get(key);
            }
        })()

        return result
    }

    put(key : number, value : Uint8Array) : void {
        this.map.update(map => {
            if (map.has(key)) {
                map.delete(key);
            }
            if (map.size >= this.capacity) {
                map.delete(map.keys().next().value);
            }
            map.set(key, value);

            return map;
        })
    }

    batchPut(values : Array<Uint8Array>) : void {
        this.map.update(map => {
            for (let i = 0; i < values.length; i++) {
                if (values[i] === undefined) {
                    continue
                }
                if (map.has(i)) {
                    map.delete(i);
                }
                if (map.size >= this.capacity) {
                    map.delete(map.keys().next().value);
                }
                map.set(i, values[i]);
            }
            return map;
        })
    }
}

export default DataCache;