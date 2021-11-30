package main

import (
	"bytes"
	"errors"
	"io"
	"math"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	EffectZeroDurationError = errors.New("zero duration not accepted")
)

var (
	effectShakeTf1 = ElasticTimeFuncGenerator(1, 1.5)
	effectShakeTf2 = ElasticTimeFuncGenerator(1.5, 2)
)

type EffectManager struct {
	render            Renderer
	shakeSeq          []float64
	shakeMaxAmplitude float64
	shakeDuration     time.Duration
	shakeFrame        int
	m                 sync.Mutex
}

func (receiver *EffectManager) applyGlobalShake(power float64, duration time.Duration) error  {
	if duration <= 0 {
		return EffectZeroDurationError
	}

	receiver.m.Lock()
	defer receiver.m.Unlock()


	seqCount := int(math.Max(float64(duration) / float64(CYCLE), 1.0)) //real cycle > CYCLE
	shFrame 	:= receiver.shakeFrame
	frameLen 	:= len(receiver.shakeSeq)

	if DEBUG_SHAKE {
		logger.Printf("set shake for %d frame with power %f \n", seqCount, power)
	}

	totalSeqCount := float64(seqCount)
	shakeAmplitude := receiver.shakeMaxAmplitude * power
	for i := shFrame; i < frameLen - 1 && seqCount > 0; i += 2 {
		tf 	:= effectShakeTf1(float64(seqCount) / totalSeqCount ) * shakeAmplitude
		tf2 := effectShakeTf2(float64(seqCount) / totalSeqCount ) * shakeAmplitude
		receiver.shakeSeq[i] 	= math.Max(math.Min(tf + receiver.shakeSeq[i], receiver.shakeMaxAmplitude), -receiver.shakeMaxAmplitude)
		receiver.shakeSeq[i+1] 	= math.Max(math.Min(tf2 + receiver.shakeSeq[i+1], receiver.shakeMaxAmplitude), -receiver.shakeMaxAmplitude)
		if DEBUG_SHAKE {
			logger.Printf("amplify shake wave, new values is : %f %f\n", receiver.shakeSeq[i], receiver.shakeSeq[i+1])
		}
		seqCount--
	}
	for seqCount > 0 {
		tf  := effectShakeTf1(float64(seqCount) / totalSeqCount ) * shakeAmplitude
		tf2 := effectShakeTf2(float64(seqCount) / totalSeqCount ) * shakeAmplitude
		receiver.shakeSeq = append(receiver.shakeSeq, tf, tf2)
		if DEBUG_SHAKE {
			logger.Printf("shake wave, values is : %f %f\n", tf, tf2)
		}
		seqCount--
	}
	receiver.shakeDuration = time.Duration(math.Max(float64(receiver.shakeDuration), float64(duration)))

	return nil
}

func (receiver *EffectManager) Execute(timeLeft time.Duration)  {
	receiver.m.Lock()
	defer receiver.m.Unlock()
	if receiver.shakeDuration > 0 {
		receiver.render.SetOffset(int(receiver.shakeSeq[receiver.shakeFrame]), int(receiver.shakeSeq[receiver.shakeFrame + 1]))
		receiver.shakeFrame += 2
		receiver.shakeDuration -= timeLeft
	} else {
		receiver.shakeDuration = 0
		receiver.shakeFrame = 0
		receiver.shakeSeq = receiver.shakeSeq[0:0]
		receiver.render.SetOffset(0, 0)
	}

	//todo other effects
}

func NewEffectManager(r Renderer) (*EffectManager, error) {
	return &EffectManager{
		render:            r,
		shakeSeq:          make([]float64, 0, 256),
		shakeMaxAmplitude: 3,
		shakeDuration:     0,
		shakeFrame:        0,
	}, nil
}

func EffectAnimNormalizeNewLine(path string, length int)  {
	for i := 0; i < length; i++ {
		path := "./sprite/" + path + "_" + strconv.Itoa(i)
		byte, err := os.ReadFile(path)
		if err == nil {
			writer, err := os.OpenFile(path, os.O_TRUNC | os.O_WRONLY, 0665)
			if err != nil {
				logger.Println(err)
			}
			if seed != 0 {
				rand.Seed(seed)
			}
			EffectNormalizeNewLine(bytes.NewBuffer(byte), writer)
			writer.Close()
		}
	}
}

func EffectAnimDisappear(path string, length int, seed int64)  {
	for i := 0; i < length; i++ {
		path := "./sprite/" + path + "_" + strconv.Itoa(i)
		byte, err := os.ReadFile(path)
		if err == nil {
			writer, err := os.OpenFile(path, os.O_TRUNC | os.O_WRONLY, 0665)
			if err != nil {
				logger.Println(err)
			}
			if seed != 0 {
				rand.Seed(seed)
			}
			EffectDisappear(bytes.NewBuffer(byte), writer, 1.0 / (float64(length - i) + 0.1))
			writer.Close()
		}
	}
}

func EffectAnimInterference(path string, length int, power float64)  {
	for i := 0; i < length; i++ {
		path := "./sprite/" + path + "_" + strconv.Itoa(i)
		byte, err := os.ReadFile(path)
		if err == nil {
			writer, err := os.OpenFile(path, os.O_TRUNC | os.O_WRONLY, 0665)
			if err != nil {
				logger.Println(err)
			}
			EffectDisappear(bytes.NewBuffer(byte), writer, power)
			writer.Close()
		}
	}
}

func EffectAnimVerFlip(path string, length int)  {
	for i := 0; i < length; i++ {
		path := "./sprite/" + path + "_" + strconv.Itoa(i)
		byte, err := os.ReadFile(path)
		if err == nil {
			writer, err := os.OpenFile(path, os.O_TRUNC | os.O_WRONLY, 0665)
			if err != nil {
				logger.Println(err)
			}
			if seed != 0 {
				rand.Seed(seed)
			}
			EffectVerFlip(bytes.NewBuffer(byte), writer)
			writer.Close()
		}
	}
}

func EffectDisappear(reader io.Reader, writer io.Writer, power float64)  {
	buf := make([]byte, 1, 1)
	for {
		rLen, _ := reader.Read(buf)
		if rLen < 1 {
			break
		}
		if buf[0] == 32 || buf[0] == 10  {
			//nope
		} else if rand.Float64() < power {
			buf[0] = 32
		}
		writer.Write(buf)
	}
}

func EffectNormalizeNewLine(reader io.Reader, writer io.Writer)  {
	buf 	:= make([]byte, 256, 256)
	all 	:= make([]byte, 0)
	for {
		rLen, err := reader.Read(buf)
		all = append(all, buf[0:rLen]...)
		if err == io.EOF {
			break
		}
	}
	parted  :=  bytes.Split(all, []byte{13, 10})
	partLen := len(parted)
	for i := 0; i < partLen / 2; i++ {
		parted[i], parted[partLen - 1 - i] = parted[partLen - 1 - i], parted[i]
	}
	writer.Write(bytes.Join(parted, []byte{10}))
}

func EffectHorFlip(reader io.Reader, writer io.Writer)  {
	buf 	:= make([]byte, 256, 256)
	all 	:= make([]byte, 0)
	for {
		rLen, err := reader.Read(buf)
		all = append(all, buf[0:rLen]...)
		if err == io.EOF {
			break
		}
	}
	parted  :=  bytes.Split(all, []byte{10})
	partLen := len(parted)
	for i := 0; i < partLen / 2; i++ {
		parted[i], parted[partLen - 1 - i] = parted[partLen - 1 - i], parted[i]
	}
	writer.Write(bytes.Join(parted, []byte{10}))
}

func EffectVerFlip(reader io.Reader, writer io.Writer)  {
	buf 	:= make([]byte, 256, 256)
	all 	:= make([]byte, 0)
	parted  := make([][]byte, 0)

	for {
		rLen, err := reader.Read(buf)
		if rLen > 0 {
			all 	= append(all, buf[0:rLen]...)
			parted 	= bytes.Split(all, []byte{10})
			pLen   := len(parted)
			if  pLen > 1 {
				if all[len(all) - 1] != 10 {
					all 	= parted[pLen 	- 1]
					parted 	= parted[:pLen 	- 1]
				} else {
					all = all[0:0]
				}
				for _, writeBuff := range parted {
					wbLen := len(writeBuff)
					for i := 0; i < wbLen / 2; i++ {
						writeBuff[i], writeBuff[wbLen - 1 - i] = writeBuff[wbLen - 1 - i], writeBuff[i]
					}
					writer.Write(writeBuff)
					writer.Write([]byte{10})
				}
			}
		}
		if err == io.EOF {
			break
		}
	}
	if lAll := len(all); lAll > 0 {
		for i := 0; i < lAll / 2; i++ {
			all[i], all[lAll - 1 - i] = all[lAll - 1 - i], all[i]
		}
		writer.Write(all)
	}
}
