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
var tokenReSimpl = regexp.MustCompile(`(\\033\[3%dm")<val>(\\033\[0m)`)
var tokenReZeros = regexp.MustCompile("0+")

var (
	SpriteExistError              = errors.New("new sprite Id exist")
	InvalidSpriteParentError      = errors.New("invalid sprite parent")
	SpriteNotFoundError           = errors.New("sprite do not exist")
	SpriteTransparentError        = errors.New("transparent must set at load")
	SpriteEmptyCustomizationError = errors.New("empty sprite customization list")

	specialSymbols = []string{"0", "\\", "3", "[", "m"}
	ErrorSprite    = NewContentSprite([]byte("!!!Error!!!"))
)

type SpriteInfo struct {
	Parent                              Spriteer
	Size                                GeoSize
	Len                                 int64 //pure len without \\blabla100500
	isTransparent, isNoClip, isAbsolute bool
}

type Spriteer interface {
	io.Writer
	fmt.Stringer
	GetInfo() *SpriteInfo
}

type CustomizeMap map[string]int

type Sprite struct {
	Buf *bytes.Buffer
	*SpriteInfo
}

func (s *Sprite) GetInfo() *SpriteInfo {
	return s.SpriteInfo
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
	box.SpriteInfo = new(SpriteInfo)
	box.Buf = new(bytes.Buffer)

	return box
}

func NewContentSprite(buffer []byte) *Sprite {

	box := new(Sprite)
	box.SpriteInfo = new(SpriteInfo)
	box.Buf = bytes.NewBuffer(buffer)
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

func AddSprite(id string, sprite *Sprite) error {
	if _, ok := sprites[id]; ok {
		return SpriteExistError
	}
	sprites[id] = sprite
	return nil
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
		var ok bool
		if sprite, ok = sprite.Parent.(*Sprite); !ok {
			return ErrorSprite, InvalidSpriteParentError
		}
	}

	if len(custom) == 0 {
		return ErrorSprite, SpriteEmptyCustomizationError
	}

	index := make([]string, 0)
	for key, _ := range custom {
		index = append(index, key)
	}
	//no sort for now, order dictating by config
	/*	sort.Slice(index, func(i, j int) bool {
		return len(index[i]) > len(index[j]) //wide first
	})*/

	spriteBuffer := sprite.String()
	if color, ok := custom["0"]; ok {
		colored := direct.Color("0", color)
		spriteBuffer = strings.ReplaceAll(spriteBuffer, "0", colored)
	}
	for _, str := range index {
		if str == "0" {
			continue
		}
		colored := direct.Color(str, custom[str])
		spriteBuffer = strings.Replace(spriteBuffer, str, colored, -1)
		reg, err := regexp.Compile(fmt.Sprintf("(%s)+", regexp.QuoteMeta(str)))
		if err == nil {
			spriteBuffer = tokenReSimpleReplaceAll(reg, spriteBuffer, colored)
		} else {
			logger.Println(err)
		}
	}

	newSprite := NewSprite()
	newSprite.Write([]byte(spriteBuffer))
	newSprite.Parent = sprite
	newSprite.Size = sprite.Size
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

func tokenReSimpleReplaceAll(token *regexp.Regexp, target string, newValue string) string {
	return token.ReplaceAllString(target, newValue)
}

func tokenReReplaceAll(token *regexp.Regexp, target string, newValue string) string {
	match := token.FindAllStringSubmatch(target, -1)
	for i, _ := range match {
		if match[i][1] == match[i][3] {
			target = strings.Replace(target, match[i][0], newValue, 1)
		}
	}
	return target
}
