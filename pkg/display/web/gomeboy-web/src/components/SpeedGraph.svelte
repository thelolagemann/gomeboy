<!--<div>
	<i style="bottom: {Math.trunc($transferRate / 1024)}px"></i>
</div>-->
<canvas id="speedChart" width="250" height="100"></canvas>

<script>
	import Game from "$lib/game";
	import {Chart,
		LineController,
		LineElement,
		PointElement,
		LinearScale, LogarithmicScale,
		registry,
		CategoryScale, Filler } from "chart.js";
	import {browser} from "$app/environment";
	import {onMount} from "svelte";
	import {get} from "svelte/store";


	let { transferRate } = Game;
	let chartCanvas;
	let speedChart;

	const initialDataLength = 60;
	const dataPointsPerUpdate = 1;
	const scrollSpeed = 1000;

	const data = {
		datasets: [{
			parsing: false,
			data: Array.from({ length: initialDataLength }, () => 0),
			fill: true,
		}],
	}

	const options = {
		animation: {
			duration: 1000,
			easing: 'linear'
		},
		elements: {
			line: {
				backgroundColor: 'blueviolet',
				fill: 'origin',
				tension: 0.125
			},
			point: {
				radius: 0
			}
		},
		scales: {
			x: {
				type: 'linear',
				ticks: {
					display: false
				},
				border: {
					display: false
				},
				grid: {
					display: false
				}
			},
			y: {
				type: 'logarithmic',
				ticks: {
					display: false
				},
				border: {
					display: false
				},
				grid: {
					display: false,
				},
				beginAtZero: true,
				title: {
					display: false
				}
			}
		},

	}

	if (browser) {
		onMount(() => {
			registry.addControllers(LineController)
			registry.addScales(LinearScale, LogarithmicScale, CategoryScale)
			registry.addElements(LineElement, PointElement)
			registry.addPlugins(Filler)
			chartCanvas = document.getElementById("speedChart").getContext("2d");

			speedChart = new Chart(chartCanvas, {
				type: 'line',
				data: data,
				options: options,
			})

			setInterval(() => {
				const newData = speedChart.data.datasets[0].data
				newData.pop()
				newData.unshift(get(transferRate));

				speedChart.update()
			}, scrollSpeed)

		});
	}
</script>

<style lang="scss">
	div {
		position: relative;

		i {
			transition: all 1s;
			left: calc(50% - 4px);
			background-color: red;
			width: 8px;
			height: 8px;
			display: block;
			border-radius: 50%;
			position: absolute;
			bottom: 0px;
		}
	}
</style>