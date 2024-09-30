package views

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/argusdusty/gofft"
	"github.com/thelolagemann/gomeboy/internal/apu"
	"image"
	"math"
	"sync"
)

const (
	numSamples    = 1024
	numChannels   = 4
	canvasWidth   = 1024
	channelHeight = 60
)

var (
	channelNames = []string{"Pulse 1", "Pulse 2", "Wave", "Noise", "Left", "Right", "Left (Filtered)", "Right (Filtered)"}
)

type Visualiser struct {
	widget.BaseWidget
	*apu.APU

	bitmaps      []*canvas.Raster
	samples      [numChannels * 2][]float32
	fftResults   [numChannels * 2][][][]byte // Buffer for precomputed FFT magnitudes
	normalizedDB [numChannels * 2][32]float64

	waveformImages    [numChannels * 2]*image.RGBA
	spectrogramImages [numChannels * 2]*image.RGBA
	dbMeterImages     [numChannels * 2]*image.RGBA

	specEnabled, dbEnabled, wavEnabled bool
}

func NewVisualiser(a *apu.APU) *Visualiser {
	v := &Visualiser{APU: a}
	v.ExtendBaseWidget(v)
	return v
}

func (v *Visualiser) CreateRenderer() fyne.WidgetRenderer {
	v.APU.Visualise(true)
	m := container.NewHBox()

	// checkbox states
	channelsEnabled := make([]bool, numChannels*2)

	for i := 0; i < numChannels*2; i++ {
		channelsEnabled[i] = true
	}

	// Bitmap canvas for each channel
	v.bitmaps = make([]*canvas.Raster, numChannels*6)
	baseContainers := [numChannels * 2]fyne.CanvasObject{}

	for ch := 0; ch < numChannels; ch++ {
		// create a base container for the channel
		b := container.NewVBox(widget.NewLabel(channelNames[ch]))
		v.waveformImages[ch] = image.NewRGBA(image.Rect(0, 0, canvasWidth, channelHeight))

		// channel output waveform & spectrogram
		r := canvas.NewRasterFromImage(v.waveformImages[ch])
		r.SetMinSize(fyne.NewSize(canvasWidth, channelHeight))
		v.bitmaps[ch*3] = r

		v.spectrogramImages[ch] = image.NewRGBA(image.Rect(0, 0, canvasWidth, channelHeight))
		rFreq := canvas.NewRasterFromImage(v.spectrogramImages[ch])
		rFreq.ScaleMode = canvas.ImageScalePixels
		rFreq.SetMinSize(fyne.NewSize(canvasWidth, channelHeight))
		v.bitmaps[(ch*3)+1] = rFreq
		v.dbMeterImages[ch] = image.NewRGBA(image.Rect(0, 0, canvasWidth, channelHeight))
		rTemp := canvas.NewRasterFromImage(v.dbMeterImages[ch])
		rTemp.SetMinSize(fyne.NewSize(canvasWidth, channelHeight))
		v.bitmaps[(ch*3)+2] = rTemp

		// add rasters to base container
		b.Add(r)
		b.Add(rFreq)
		b.Add(rTemp)
		baseContainers[ch] = b

		b2 := container.NewVBox(widget.NewLabel(channelNames[ch+4]))
		v.waveformImages[ch+4] = image.NewRGBA(image.Rect(0, 0, canvasWidth, channelHeight))
		r2 := canvas.NewRasterFromImage(v.waveformImages[ch+4])
		r2.SetMinSize(fyne.NewSize(canvasWidth, channelHeight))
		v.bitmaps[(ch*3)+12] = r2
		v.spectrogramImages[ch+4] = image.NewRGBA(image.Rect(0, 0, canvasWidth, channelHeight))
		r2Freq := canvas.NewRasterFromImage(v.spectrogramImages[ch+4])
		r2Freq.SetMinSize(fyne.NewSize(canvasWidth, channelHeight))
		v.bitmaps[(ch*3)+13] = r2Freq
		v.dbMeterImages[ch+4] = image.NewRGBA(image.Rect(0, 0, canvasWidth, channelHeight))
		r2Temp := canvas.NewRasterFromImage(v.dbMeterImages[ch+4])
		r2Temp.SetMinSize(fyne.NewSize(canvasWidth, channelHeight))
		v.bitmaps[(ch*3)+14] = r2Temp
		b2.Add(r2)
		b2.Add(r2Freq)
		b2.Add(r2Temp)
		baseContainers[ch+4] = b2
	}

	v.wavEnabled, v.specEnabled, v.dbEnabled = true, true, true

	updateVisualiser := func(channel int) {
		if channelsEnabled[channel] {
			if v.wavEnabled {
				v.bitmaps[channel*3].Show()
			} else {
				v.bitmaps[channel*3].Hide()
			}
			if v.specEnabled {
				v.bitmaps[channel*3+1].Show()
			} else {
				v.bitmaps[channel*3+1].Hide()
			}
			if v.dbEnabled {
				v.bitmaps[channel*3+2].Show()
			} else {
				v.bitmaps[channel*3+2].Hide()
			}
		} else {
			// Hide all if the channel is disabled
			v.bitmaps[channel*3].Hide()
			v.bitmaps[channel*3+1].Hide()
			v.bitmaps[channel*3+2].Hide()
		}
	}

	updateAllVisualisers := func() {
		for ch := 0; ch < numChannels*2; ch++ {
			updateVisualiser(ch)
		}
	}

	optionsContainer := container.NewVBox(widget.NewLabel("Options"))
	waveformCheck := widget.NewCheck("Waveform", func(b bool) {
		v.wavEnabled = !v.wavEnabled
		updateAllVisualisers()
	})
	waveformCheck.Checked = true
	spectrogramCheck := widget.NewCheck("Spectrogram", func(b bool) {
		v.specEnabled = !v.specEnabled
		updateAllVisualisers()
	})
	spectrogramCheck.Checked = true
	dbMeterCheck := widget.NewCheck("dB Meter", func(b bool) {
		v.dbEnabled = !v.dbEnabled
		updateAllVisualisers()
	})
	dbMeterCheck.Checked = true
	optionsContainer.Add(waveformCheck)
	optionsContainer.Add(spectrogramCheck)
	optionsContainer.Add(dbMeterCheck)

	channels := container.NewVBox(baseContainers[:4]...)
	outputs := container.NewVBox(baseContainers[4:]...)
	m.Add(optionsContainer)

	m.Add(channels)
	m.Add(outputs)

	m.Resize(fyne.NewSize(1400, 600))

	return widget.NewSimpleRenderer(m)
}

func (v *Visualiser) Refresh() {
	d := v.APU.WavData()
	b := v.APU.AnalogOutput()
	copy(v.samples[0:4], d[:])
	copy(v.samples[4:8], b[:])
	if v.specEnabled {
		v.precomputeFFT()
	}
	if v.dbEnabled {
		v.computeRMS()
	}
	for i := 0; i < numChannels; i++ {
		if len(d[i]) >= numSamples {
			if v.wavEnabled {
				v.updateWaveformBitmap(i)
			}
			if v.specEnabled {
				v.updateSpectrogramBitmap(i)
			}
			if v.dbEnabled {
				v.updateDBMeterBitmap(i)
			}
		}
		if len(b[i]) >= numSamples {
			if v.wavEnabled {
				v.updateWaveformBitmap(i + 4)
			}
			if v.specEnabled {
				v.updateSpectrogramBitmap(i + 4)
			}
			if v.dbEnabled {
				v.updateDBMeterBitmap(i + 4)
			}
		}
	}
}

const lineWidth = 2

func (v *Visualiser) updateWaveformBitmap(channel int) {
	img := v.waveformImages[channel]
	copy(temp, blackBar)
	for x := 0; x < canvasWidth; x++ {
		sampleIndex := (x * numSamples) / canvasWidth
		nextSampleIndex := ((x + 1) * numSamples) / canvasWidth

		amplitude := v.getAmplitude(channel, sampleIndex)
		nextAmplitude := v.getAmplitude(channel, nextSampleIndex)

		yPosition := int((channelHeight / 2) - (amplitude * (channelHeight / 2)))
		nextYPosition := int((channelHeight / 2) - (nextAmplitude * (channelHeight / 2)))

		// Draw the line in the image buffer
		for y := min(yPosition-lineWidth/2, nextYPosition-lineWidth/2); y <= max(yPosition+lineWidth/2, nextYPosition+lineWidth/2); y++ {
			if y >= 0 && y < channelHeight {
				temp[y*img.Stride+x*4+1] = 0xff // black and alpha so only need to flip green
			}
		}
	}
	copy(img.Pix, temp)
	v.bitmaps[channel*3].Refresh()
}

var (
	blackBar []byte
	temp     []byte
)

func init() {
	blackBar = make([]byte, canvasWidth*channelHeight*4)
	temp = make([]byte, len(blackBar))
}

func (v *Visualiser) updateSpectrogramBitmap(channel int) {
	img := v.spectrogramImages[channel]

	fftResults := v.fftResults[channel]
	fftLen := len(fftResults)

	for x := 0; x < canvasWidth; x++ {
		fftIndex := (x * fftLen) / canvasWidth
		if fftIndex >= fftLen {
			continue
		}
		fftResult := fftResults[fftIndex]
		freqLen := len(fftResult)

		for y := 0; y < channelHeight; y++ {
			frequencyBin := (y << 8) / channelHeight
			if frequencyBin >= freqLen {
				continue
			}

			c := fftResult[frequencyBin]
			// Calculate the index in the Pix slice for the pixel at (x, y)
			index := y*img.Stride + x<<2
			copy(img.Pix[index:], c)
		}
	}
	v.bitmaps[channel*3+1].Refresh()
}

// Get amplitude for the given channel and sample index
func (v *Visualiser) getAmplitude(channel int, sampleIndex int) float32 {
	if sampleIndex < len(v.samples[channel]) {
		return v.samples[channel][sampleIndex]
	}
	return 0.0
}

const (
	numFFTWindowSize = 512 // FFT window size
	overlapFactor    = 1
)

func mapMagnitudeToColor(magnitude float64) []byte {
	var by = make([]byte, 4)

	// Normalize the magnitude to a range between 0 and 1
	normalizedMagnitude := math.Min(1.0, magnitude)

	r, g, b := uint8(0), uint8(0), uint8(0)

	if normalizedMagnitude < 0.2 {
		// Low magnitudes (dark blue to light blue)
		b = uint8(255 * (normalizedMagnitude / 0.2))
	} else if normalizedMagnitude < 0.4 {
		// Blue to Green
		g = uint8(255 * ((normalizedMagnitude - 0.2) / 0.2))
		b = uint8(255 - g)
	} else if normalizedMagnitude < 0.6 {
		// Green to Yellow
		r = uint8(255 * ((normalizedMagnitude - 0.4) / 0.2))
		g = 255
	} else if normalizedMagnitude < 0.8 {
		// Yellow to Orange
		r = 255
		g = uint8(255 - (255 * ((normalizedMagnitude - 0.6) / 0.2)))
	} else {
		// High magnitudes (Orange to Red)
		r = 255
	}
	by[0] = r
	by[1] = g
	by[2] = b
	by[3] = 255
	return by
}

func (v *Visualiser) precomputeFFT() {
	var wg sync.WaitGroup
	var mu sync.Mutex

	for ch := 0; ch < numChannels*2; ch++ {
		wg.Add(1)
		go func(ch int, windows []float32) {
			var channelFFTResults [][][]byte

			// Slide over the samples with a window size of numFFTWindowSize, with some overlap
			for i := 0; i+numFFTWindowSize <= len(windows); i += numFFTWindowSize / overlapFactor {
				window := windows[i : i+numFFTWindowSize]
				fft64 := make([]float64, len(window))
				for in, f := range window {
					fft64[in] = float64(f)
				}

				_ = gofft.FFT(gofft.Float64ToComplex128Array(fft64)) // Perform FFT

				// Store the magnitude of each frequency bin
				magnitudes := make([][]byte, numFFTWindowSize/2) // Only need first half of FFT
				for j := range magnitudes {
					magnitudes[j] = mapMagnitudeToColor(fft64[j])
				}

				channelFFTResults = append(channelFFTResults, magnitudes)
			}
			mu.Lock()
			v.fftResults[ch] = channelFFTResults
			mu.Unlock()
			wg.Done()
		}(ch, v.samples[ch])
	}

	wg.Wait()
}

const decayRate = 0.9
const riseRate = 0.4

func (v *Visualiser) computeRMS() {
	for i := 0; i < numChannels*2; i++ {
		if len(v.samples[i]) < numSamples {
			continue
		}
		for r := 0; r < 32; r++ {
			rms := calculateRMS(v.samples[i][r*32 : r*32+32])
			db := calculateDB(rms)

			// Normalize dB to a range between 0 and 1, considering -60 dB to 0 dB
			maxDB := 0.0   // Reference max dB value (0 dB)
			minDB := -60.0 // Reference min dB value (-60 dB)
			normalizedDB := (db - minDB) / (maxDB - minDB)

			// Apply decay function: blend previous dB with the new one
			v.normalizedDB[i][r] = applyDecay(v.normalizedDB[i][r], normalizedDB, decayRate)
		}
	}
}

// Apply decay to the previous dB value
func applyDecay(previousDB, newDB, decayRate float64) float64 {
	if newDB > previousDB {
		// If the new dB is higher, update twice as fast
		return previousDB*(1.0-riseRate) + newDB*riseRate
	}
	// Apply decay if new dB is lower
	return previousDB*decayRate + newDB*(1.0-decayRate)
}

func calculateRMS(samples []float32) float64 {
	if len(samples) == 0 {
		return 0.0
	}

	firstSample := samples[0]
	allSame := true
	for _, sample := range samples {
		if sample != firstSample {
			allSame = false
			break
		}
	}

	if allSame {
		return 0.0 // Effectively silent when all samples are the same
	}

	var sum float64
	for _, sample := range samples {
		sum += float64(sample) * float64(sample)
	}

	mean := sum / float64(len(samples))
	return math.Sqrt(mean)
}

func calculateDB(rms float64) float64 {
	const silenceThreshold = 1e-9

	if rms <= silenceThreshold {
		return -60.0
	}

	db := 20 * math.Log10(rms)

	if db < -60.0 {
		db = -60.0
	}
	if db > 0.0 {
		db = 0.0
	}

	return db
}

const (
	barWidth      = 28
	spacing       = 4
	totalBarWidth = barWidth + spacing
)

func (v *Visualiser) updateDBMeterBitmap(channel int) {
	img := v.dbMeterImages[channel]
	copy(temp, blackBar)

	for barIndex := 0; barIndex < 32; barIndex++ {
		// Fix the calculation for bar width, ensuring it is consistent
		startX := barIndex << 5
		endX := startX + barWidth

		barHeight := int(v.normalizedDB[channel][barIndex] * float64(channelHeight))
		startY := channelHeight - barHeight

		// Set the pixels for the bar
		for x := startX; x < endX; x++ {
			for y := startY; y < channelHeight; y++ {
				index := y*img.Stride + (x << 2)
				temp[index+1] = 0xff
				temp[index+2] = 0xff
			}
		}
	}
	copy(img.Pix, temp)
	v.bitmaps[channel*3+2].Refresh()
}
