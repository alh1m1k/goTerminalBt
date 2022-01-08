package main

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	direct "github.com/buger/goterm"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	_ "unicode/utf8"
)

var sprites map[string]*Sprite = make(map[string]*Sprite, 20)
var tokenRe = regexp.MustCompile(`<(?P<tag>\w+?)>(?P<val>\S+?)</(?P<tg>\w+)>`)

var (
	SpriteExistError       = errors.New("new sprite Id exist")
	SpriteNotFoundError    = errors.New("sprite do not exist")
	SpriteTransparentError = errors.New("transparent must set at load")
)

var (
	ErrorSprite = NewContentSprite([]byte("!!!Error!!!"))
)

type SizeI struct {
	X, Y int
}

type Spriteer interface {
	io.Writer
	fmt.Stringer
	GetWH() GeoSize
}

type CustomizeMap map[string]int

type Sprite struct {
	Parent                  *Sprite
	Buf                     *bytes.Buffer
	Size                    GeoSize
	isTransparent, isNoClip bool
}

func (s *Sprite) GetWH() GeoSize {
	return s.Size
}

func (s *Sprite) Write(p []byte) (int, error) {
	return s.Buf.Write(p)
}

func (s *Sprite) String() (out string) {
	return s.Buf.String()
}

func (s *Sprite) CalculateSize() {
	s.Size = GeoSizeOf(s.Buf.String())
}

func NewSprite() *Sprite {

	box := new(Sprite)
	box.Parent = nil
	box.Buf = new(bytes.Buffer)
	box.isTransparent = false

	return box
}

func NewContentSprite(buffer []byte) *Sprite {

	box := new(Sprite)
	box.Parent = nil
	box.Buf = bytes.NewBuffer(buffer)
	box.isTransparent = false
	box.CalculateSize()

	return box
}

//todo remove it
func GetSprite(id string, load bool, processTransparent bool) (*Sprite, error) {
	if sprite, ok := sprites[id]; ok {
		return sprite, nil
	} else if load == false {
		return nil, SpriteNotFoundError
	}
	buffer, err := loadSprite(id)
	if err != nil {
		return nil, err
	}
	sprite := NewSprite()

	if processTransparent {
		TruncateSpaces(bytes.NewReader(buffer), sprite)
		sprite.isTransparent = true
		sprite.Size = GeoSizeOf(string(buffer))
	} else {
		sprite.Write(buffer)
		sprite.CalculateSize()
		sprite.isTransparent = false
	}
	sprites[id] = sprite

	return sprite, nil
}

func AddSprite(id string, sprite *Sprite) (*Sprite, error) {
	if _, ok := sprites[id]; ok {
		return nil, SpriteExistError
	}
	sprites[id] = sprite
	return sprite, nil
}

func LoadSprite2(path string, processTransparent bool) (*Sprite, error) {
	buffer, err := loadSprite(path)
	if err != nil {
		return ErrorSprite, err
	}
	sprite := NewSprite()
	if processTransparent {
		TruncateSpaces(bytes.NewReader(buffer), sprite)
		sprite.isTransparent = true
		sprite.Size = GeoSizeOf(string(buffer))
	} else {
		sprite.Write(buffer)
		sprite.isTransparent = false
		sprite.CalculateSize()
	}
	return sprite, nil
}

func GetSprite2(id string) (*Sprite, error) {
	if sprite, ok := sprites[id]; ok {
		return sprite, nil
	} else {
		return ErrorSprite, SpriteNotFoundError
	}
}

func IsCustomizedSpriteVer(sprite *Sprite) bool {
	if sprite.Parent != nil {
		return true
	} else {
		return false
	}
}

func CustomizeSprite(sprite *Sprite, custom CustomizeMap) (*Sprite, error) {
	if sprite.Parent != nil { //customize only base sprite
		sprite = sprite.Parent
	}
	spriteBuffer := sprite.String()
	match := tokenRe.FindAllStringSubmatch(spriteBuffer, -1)
	for i, _ := range match {
		if match[i][1] == match[i][3] {
			if color, ok := custom[match[i][1]]; ok {
				colored := direct.Color(match[i][2], color)
				spriteBuffer = strings.Replace(spriteBuffer, match[i][0], colored, 1)
			}
		}
	}
	newSprite := NewSprite()
	newSprite.Write([]byte(spriteBuffer))
	newSprite.Parent = sprite
	return newSprite, nil
}

func SwitchSprite(new, old Spriteer) {
	manager, _ := getAnimationManager()
	if new == old && new != nil {
		switch new.(type) {
		case *Animation:
			animation := new.(*Animation)
			animation.Reset()
			if animation.Manager == nil {
				manager.Add(animation)
			}
		default:

		}
		return
	}
	if new != nil {
		switch new.(type) {
		case *Animation:
			animation := new.(*Animation)
			animation.Reset()
			if animation.Manager == nil {
				manager.Add(animation)
			}
		default:

		}
	}
	if old != nil {
		switch old.(type) {
		case *Animation:
			animation := old.(*Animation)
			if animation.Manager != nil {
				animation.Manager.Remove(animation)
			}
		default:

		}
	}
}

func CopySprite(spriteer Spriteer) Spriteer {
	switch spriteer.(type) {
	case *Animation:
		return spriteer.(*Animation).Copy()
	case *Composition:
		return spriteer.(*Composition).Copy()
	default:
		//skip copy if it is just buffer
		return spriteer
	}
}

func TruncateSpaces(reader io.Reader, writer io.Writer) {
	buf := make([]byte, 1, 1)
	spaceCounter := 0
	for {
		_, err := reader.Read(buf)
		if err == io.EOF {
			break
		}
		if buf[0] == 32 {
			spaceCounter++
		} else if spaceCounter > 0 {
			fmt.Fprintf(writer, "\033[%dC", spaceCounter)
			writer.Write(buf)
			spaceCounter = 0
		} else {
			writer.Write(buf)
		}
	}
}

func GeoSizeOf(str string) GeoSize {
	strings := strings.Split(str, "\n")
	size := GeoSize{}
	size.W, size.H = 0, len(strings)
	for i := 0; i < size.H; i++ {
		size.W = maxInt(size.W, len(strings[i]))
	}
	return size
}

func customizedSpriteName(familyId string, customizeMap CustomizeMap) string {
	return familyId + "-" + hashCustomizeMap(customizeMap)
}

func hashCustomizeMap(customizeMap CustomizeMap) string {

	index := make([]string, 0)
	for key, _ := range customizeMap {
		index = append(index, key)
	}
	sort.Strings(index)
	hash := md5.New()

	for _, key := range index {
		hash.Write([]byte(key))
		hash.Write([]byte(strconv.Itoa(customizeMap[key])))
	}

	return string(hash.Sum(nil))
}
