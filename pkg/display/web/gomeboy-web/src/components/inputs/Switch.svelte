<div>
	<input type="checkbox" id={name} {name} bind:checked={value}/>
	<label for={name}>
		{text}
		<slot></slot>
	</label>
</div>

<script>
	export let text;
	export let name;
	export let value;
</script>

<style lang="scss">
	@use "sass:color";

	// Variables
	$bg-disabled-color: color.scale(gray, $whiteness: +25%);
	$bg-enabled-color: color.scale(blueviolet, $whiteness: +25%);
	$lever-disabled-color: #fff;
	$lever-enabled-color: blueviolet;

	div {
		display: inline-block;
		position: relative;
		font-size: 16px;
		line-height: 24px;

		input {
			position: absolute;
			top: 0;
			left: 0;
			width: 36px;
			height: 20px;
			opacity: 0;
			z-index: 0;
		}

		label {
			color: color.scale(gray, $whiteness: +90%);
			cursor: pointer;
			display: block;
			font-size: 14px;
			padding: 0 0 0 44px;

			&:before {
				 background-color: $bg-disabled-color;
				 content: '';
				 position: absolute;
				 top: 5px;
				 left: 0;
				 width: 36px;
				 height: 14px;
				 border-radius: 14px;
				 z-index: 1;
				 transition: background-color 0.28s cubic-bezier(.4, 0, .2, 1);
			 }

			&:after {
				 content: '';
				 position: absolute;
				 top: 2px;
				 left: 0;
				 width: 20px;
				 height: 20px;
				 background-color: $lever-disabled-color;
				 border-radius: 14px;
				 box-shadow: 0 2px 2px 0 rgba(0, 0, 0, .14),0 3px 1px -2px rgba(0, 0, 0, .2), 0 1px 5px 0 rgba(0, 0, 0, .12);
				 z-index: 2;
				 transition: all 0.28s cubic-bezier(.4, 0, .2, 1);
				 transition-property: left, background-color;
			 }
		}

		input:checked + label {
			&:before {
				 background-color: $bg-enabled-color;
			}

			&:after {
				 left: 16px;
				 background-color: $lever-enabled-color;
			}
		}
	}
</style>