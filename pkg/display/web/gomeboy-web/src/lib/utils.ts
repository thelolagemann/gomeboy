import {faker} from "@faker-js/faker";
import { browser } from "$app/environment";


export function testBit(value: number, bit: number) : boolean {
    return (value & (1 << bit)) !== 0
}

/**
 * Format bytes as human-readable text.
 *
 * @param bytes Number of bytes.
 * @param si True to use metric (SI) units, aka powers of 1000. False to use
 *           binary (IEC), aka powers of 1024.
 * @param dp Number of decimal places to display.
 *
 * @return Formatted string.
 */
export function humanFileSize(bytes, si=false, dp=1) {
    const thresh = si ? 1000 : 1024;

    if (Math.abs(bytes) < thresh) {
        return bytes + ' B';
    }

    const units = si
        ? ['kB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']
        : ['KiB', 'MiB', 'GiB', 'TiB', 'PiB', 'EiB', 'ZiB', 'YiB'];
    let u = -1;
    const r = 10**dp;

    do {
        bytes /= thresh;
        ++u;
    } while (Math.round(Math.abs(bytes) * r) / r >= thresh && u < units.length - 1);


    return bytes.toFixed(dp) + ' ' + units[u];
}

export function getOS(userAgent: string) : string {
    const macPlatforms = ['Macintosh', 'MacIntel', 'MacPPC', 'Mac68K'];
    const windowsPlatforms = ['Win32', 'Win64', 'Windows', 'WinCE'];
    const iosPlatforms = ['iPhone', 'iPad', 'iPod'];

    let os = 'unknown'
    if (macPlatforms.some(v => userAgent.includes(v))) {
        os = 'Mac'
    } else if (windowsPlatforms.some(v => userAgent.includes(v))) {
        os = 'Windows'
    } else if (iosPlatforms.some(v => userAgent.includes(v))) {
        os = 'iOS'
    } else if (/Android/.test(userAgent)) {
        os = 'Android'
    } else if (/Linux/.test(userAgent)) {
        os = 'Linux'
    }

    return os
}

/**
 * Get the preferred username from localStorage if
 * it exists, otherwise generates a random username
 * using faker-js
 */
export function getUsername(): string {
    let username = localStorage.getItem('username')
    if (username === null) {
        let adverb = faker.word.adverb(5), adjective = faker.word.adjective(5), name = faker.person.firstName()
        username = adverb.charAt(0)
            .toUpperCase()
            .concat(adverb.slice(1))
            .concat(adjective.charAt(0)
                .toUpperCase()
                .concat(adjective.slice(1)))
            .concat(name)
    }

    return username
}

/**
 * Set the preferred username in localStorage
 * @param username
 */
export function setUsername(username: string) {
    localStorage.setItem('username', username)
}