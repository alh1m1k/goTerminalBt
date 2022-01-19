package main

import (
	"errors"
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"os"
	"time"
)

var (
	soundInited = false

	FileRegisteredError = errors.New("file already registered")
	FileNotFoundError   = errors.New("file not found")
	SoundNotReadyError  = errors.New("sound not ready")
)

type SoundInfo struct {
	Key     string
	Path    string
	Handler *os.File
	beep.Format
	*beep.Buffer
	Stream        beep.StreamSeekCloser
	Loaded, Ready bool
}

type SoundManager struct {
	sounds map[string]*SoundInfo
}

func (receiver *SoundManager) Register(key string, path string, prefeth bool) error {
	if _, ok := receiver.sounds[key]; ok {
		return FileRegisteredError
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	streamer, format, err := receiver.decode(f, path)
	if err != nil {
		return err
	}
	info := SoundInfo{
		Key:     key,
		Path:    path,
		Handler: f,
		Format:  format,
		Buffer:  nil,
		Stream:  streamer,
	}
	if prefeth {
		info.Buffer = beep.NewBuffer(format)
		info.Buffer.Append(info.Stream)
		info.Stream.Close()
		info.Handler.Close()
		info.Stream = nil
		info.Handler = nil
		info.Loaded = true
	}
	logger.Println(info.Format)
	receiver.sounds[key] = &info
	info.Ready = true
	return nil
}

func (receiver *SoundManager) Play(key string) error {
	if soundInfo, ok := receiver.sounds[key]; ok {
		if !soundInfo.Ready {
			return SoundNotReadyError
		}
		if soundInfo.Loaded {
			speaker.Play(soundInfo.Buffer.Streamer(0, soundInfo.Buffer.Len()))
		} else {
			speaker.Play(soundInfo.Stream)
		}
	} else {
		return FileNotFoundError
	}
	return nil
}

func (receiver *SoundManager) Background(key string) error {
	if soundInfo, ok := receiver.sounds[key]; ok {
		if !soundInfo.Ready {
			return SoundNotReadyError
		}
		if soundInfo.Loaded {
			speaker.Play(soundInfo.Buffer.Streamer(0, soundInfo.Buffer.Len()))
		} else {
			speaker.Play(soundInfo.Stream)
		}
	} else {
		return FileNotFoundError
	}
	return nil
}

func (receiver *SoundManager) decode(file *os.File, path string) (streamer beep.StreamSeekCloser, format beep.Format, err error) {
	return mp3.Decode(file)
}

func NewSoundManager() (*SoundManager, error) {
	if !soundInited {
		format := beep.Format{}
		format.SampleRate = 44100
		err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/30))
		if err != nil {
			return nil, err
		}
	}
	return &SoundManager{
		sounds: make(map[string]*SoundInfo),
	}, nil
}
