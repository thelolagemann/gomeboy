class Cache {
    capacity: number;
    map: Map<number, Uint8Array>;
    hits: number;

	constructor(capacity: number) {
        this.capacity = capacity;
        this.map = new Map();
        this.hits = 0;
    }

    get(key: number) : Uint8Array | null{
        if (this.map.has(key)) {
            const value = this.map.get(key);
            this.map.delete(key);
            this.map.set(key, value);
            this.hits++
            return value;
        }
        return null;
    }

    put(key : number, value : Uint8Array) : void {
        if (this.map.has(key)) {
            this.map.delete(key);
        } else if (this.map.size >= this.capacity) {
            this.map.delete(this.map.keys().next().value);
        }
        this.map.set(key, value);
    }

    averageSize() : number{
        return Math.trunc(this.byteSize() / this.map.size);
    }

    byteSize() : number {
        let size = 0;
        for (const [_, value] of this.map) {
            size += value.byteLength;
        }
        return size;
    }

    length() : number {
        return this.map.size;
    }

}

export default Cache;