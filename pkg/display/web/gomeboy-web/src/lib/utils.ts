export function testBit(value: number, bit: number) : boolean {
    return (value & (1 << bit)) !== 0
}