import { slide } from 'svelte/transition';

export function fadeSlide(node, options) {
    const slideTrans = slide(node, options)
    return {
        duration: options.duration,
        css: t => `
            ${slideTrans.css(t)}
            opacity: ${t}`
    }
}