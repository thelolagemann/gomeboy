export const compressionLevelMap : {[key: number]: string} = {
    1: "Fastest",
    2: "Faster",
    3: "Default",
    4: "Slowest",
}

export const keyMap : {[key: string]: number} = {
    "a": 0,
    "b": 1,
    "Backspace": 2,
    "Enter": 3,
    "ArrowRight": 4,
    "ArrowLeft": 5,
    "ArrowUp": 6,
    "ArrowDown": 7,
}

export const dpadKeyMap = {
    "ArrowRight": "right",
    "ArrowLeft": "left",
    "ArrowUp": "up",
    "ArrowDown": "down",
    "a": "a",
    "b": "b"
}

export enum Button {
    A,
    B,
    Select,
    Start,
    Right,
    Left,
    Up,
    Down,
}

export const ButtonClassMap: {[key: Button]: string} = {
    [Button.A]: "a",
    [Button.B]: "b",
    [Button.Select]: "select",
    [Button.Start]: "start",
    [Button.Right]: "right",
    [Button.Left]: "left",
    [Button.Up]: "up",
    [Button.Down]: "down",
}

export const ButtonIconMap: {[key: Button]: string} = {
    [Button.A]: "A",
    [Button.B]: "B",
    [Button.Select]: "Select",
    [Button.Start]: "Start",
    [Button.Right]: "arrow_drop_down",
    [Button.Left]: "arrow_drop_down",
    [Button.Up]: "arrow_drop_down",
    [Button.Down]: "arrow_drop_down",
}

