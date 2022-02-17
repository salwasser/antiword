package main

import (
	"antiword/dictionary"
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/inpututil"
	"image/color"
	"log"
	"math"
	"math/rand"
	"strings"
	"time"
)

const (
	UNKNOWN = iota
	RIGHTLETTERWRONGPLACE
	RIGHTLETTERRIGHTPLACE
	WRONGLETTER
	BORDERCOLOR
)

var ColorStateMap = map[int]color.RGBA{
	UNKNOWN:               {R: 100, G: 100, B: 100, A: 255},
	RIGHTLETTERWRONGPLACE: {R: 255, G: 220, B: 0, A: 255},
	RIGHTLETTERRIGHTPLACE: {R: 0, G: 200, B: 0, A: 255},
	WRONGLETTER:           {R: 188, G: 0, B: 0, A: 255},
	BORDERCOLOR:           {R: 208, G: 208, B: 208, A: 255},
}

type ColorfulLetter struct {
	letter      uint8
	state       int
	borderWidth float64
	thickness   float64
}

func (cl *ColorfulLetter) lineBorderWidth() float64 {
	if cl.borderWidth > 0.0 {
		return cl.borderWidth
	} else {
		return 4.0
	}
}

func (cl *ColorfulLetter) lineThickness() float64 {
	if cl.thickness > 0.0 {
		return cl.thickness
	} else {
		return 4.0
	}
}

func (cl *ColorfulLetter) Draw(screen *ebiten.Image, x1 float64, y1 float64, x2 float64, y2 float64) {
	ebitenutil.DrawRect(screen,
		x1+cl.lineBorderWidth()+cl.lineThickness(),
		y1+cl.lineBorderWidth()+cl.lineThickness(),
		((x2-cl.lineBorderWidth())-cl.lineThickness())-(x1+cl.lineBorderWidth()+cl.lineThickness()),
		((y2-cl.lineBorderWidth())-cl.lineThickness())-(y1+cl.lineBorderWidth()+cl.lineThickness()),
		ColorStateMap[cl.state])

	for t := float64(0); t < cl.lineThickness(); t = t + 1.0 {
		//Horizontal top:
		ebitenutil.DrawLine(screen, x1+cl.lineBorderWidth()+t, y1+cl.lineBorderWidth()+t, (x2-cl.lineBorderWidth())-t, y1+cl.lineBorderWidth()+t, ColorStateMap[BORDERCOLOR])
		//Horizontal bottom:
		ebitenutil.DrawLine(screen, x1+cl.lineBorderWidth()+t, (y2-cl.lineBorderWidth())-t, (x2-cl.lineBorderWidth())-t, (y2-cl.lineBorderWidth())-t, ColorStateMap[BORDERCOLOR])
		//Vertical left:
		ebitenutil.DrawLine(screen, x1+cl.lineBorderWidth()+t, y1+cl.lineBorderWidth()+t, x1+cl.lineBorderWidth()+t, (y2-cl.lineBorderWidth())-t, ColorStateMap[BORDERCOLOR])
		//Vertical right:
		ebitenutil.DrawLine(screen, (x2-cl.lineBorderWidth())-t, y1+cl.lineBorderWidth()+t, (x2-cl.lineBorderWidth())-t, (y2-cl.lineBorderWidth())-t, ColorStateMap[BORDERCOLOR])
	}
	if cl.letter != 0 {
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%c", cl.letter), int(x1+(x2-x1)/4.0), int(y1+(y2-y1)/4.0))
	}
}

type PositionAndClaim struct {
	pos   int
	claim bool
	bound bool
}

type AntiwordGame struct {
	playtrix       [][]ColorfulLetter
	keys           map[uint8]uint8
	dict           []string
	dictMap        map[string]bool
	word           string
	wordMap        map[uint8][]PositionAndClaim
	entryWord      int
	entryLetter    int
	enforcementMap map[int]map[uint8]bool
}

func (awg *AntiwordGame) GetWordSize() int {
	return 5
}

func (awg *AntiwordGame) GetWidth() int {
	return 360
}

func (awg *AntiwordGame) GetHeight() int {
	return 720
}

func (awg *AntiwordGame) PlaytrixBorderColor() (color color.RGBA) {
	color.G = 255
	color.R = 255
	color.B = 255
	color.A = 255
	return
}

func (awg *AntiwordGame) BackgroundColor() (color color.RGBA) {
	return
}

func (awg *AntiwordGame) GetPlaySpaceEndRatio() float64 {
	return 0.70
}

func (awg *AntiwordGame) GetKeyBoardStartRatio() float64 {
	return 0.20
}

func (awg *AntiwordGame) Reset(debug []string) {

	//Load the word list as a map.
	awg.dictMap = make(map[string]bool)
	for i := range awg.dict {
		awg.dictMap[strings.ToUpper(awg.dict[i])] = true
	}

	//Create a map to hold the state of each input key.
	awg.keys = make(map[uint8]uint8)

	awg.playtrix = [][]ColorfulLetter{}
	awg.playtrix = append(awg.playtrix, make([]ColorfulLetter, awg.GetWordSize(), awg.GetWordSize()))
	awg.entryWord = 0

	//Create a map to hold the remaining legal letters for each game board column.
	awg.enforcementMap = make(map[int]map[uint8]bool)
	for i := 0; i < awg.GetWordSize(); i++ {
		awg.enforcementMap[i] = make(map[uint8]bool)
		for j := uint8('A'); j <= uint8('Z'); j++ {
			awg.enforcementMap[i][j] = true
		}
	}

	//Pick the word, sanitize the input, etc.
	rand.Seed(time.Now().UnixNano())
	wordIdx := int(rand.Float64() * float64(len(awg.dict)))
	if debug == nil {
		awg.word = strings.ToUpper(awg.dict[wordIdx])
	} else {
		awg.word = strings.ToUpper(awg.dict[0])
		fmt.Println("Word:", awg.word)
	}

	//Create data structure permitting guessed letters to stake a claim on
	//specific letters in the secret word.
	awg.wordMap = make(map[uint8][]PositionAndClaim)
	for i := range awg.word {
		if awg.wordMap[awg.word[i]] == nil {
			awg.wordMap[awg.word[i]] = []PositionAndClaim{{pos: i}}
		} else {
			awg.wordMap[awg.word[i]] = append(awg.wordMap[awg.word[i]], PositionAndClaim{pos: i})
		}
	}

	if debug != nil {
		for i := range debug {
			debug[i] = strings.ToUpper(debug[i])
			for i >= len(awg.playtrix) {
				awg.playtrix = append(awg.playtrix, make([]ColorfulLetter, awg.GetWordSize(), awg.GetWordSize()))
			}
			for j := 0; j < awg.GetWordSize(); j++ {
				awg.playtrix[i][j].letter = debug[i][j]
			}
			awg.EnterPressed()
		}
	}
}

func (awg *AntiwordGame) Layout(outsideWidth int, outsideHeight int) (screenWidth int, screenHeight int) {
	return outsideWidth, outsideHeight
}

func (awg *AntiwordGame) Draw(screen *ebiten.Image) {
	playtrixXStepValue := awg.GetWidth() / awg.GetWordSize()
	playtrixYStepValue := math.Min(float64(awg.GetHeight())*awg.GetPlaySpaceEndRatio()/float64(len(awg.playtrix)), float64(awg.GetHeight())*awg.GetPlaySpaceEndRatio()/6.0)
	for y := range awg.playtrix {
		xIdx := 0
		for x := 0; x < awg.GetWidth(); x += playtrixXStepValue {
			awg.playtrix[y][xIdx].Draw(screen, float64(x), float64(y)*playtrixYStepValue, float64(x+playtrixXStepValue), float64(y+1)*playtrixYStepValue)
			xIdx++
		}
	}

	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("Score: %d.", len(awg.playtrix)), 0, int(float64(awg.GetHeight())*awg.GetPlaySpaceEndRatio()))

	kbXStepValue := float64(awg.GetWidth() / 9)
	kbYStepValue := float64(awg.GetHeight()) * awg.GetKeyBoardStartRatio() / 4.0
	kbYStartValue := float64(awg.GetHeight()) - float64(awg.GetHeight())*awg.GetKeyBoardStartRatio()

	letterCode := uint8('A')
	rowCount := 0
	for y := kbYStartValue; y < float64(awg.GetHeight()); y = y + kbYStepValue {
		if rowCount < 3 {
			for x := float64(0); x < float64(awg.GetWidth()); x = x + kbXStepValue {
				cl := ColorfulLetter{
					letter:      letterCode,
					state:       int(awg.keys[letterCode]),
					borderWidth: 8,
					thickness:   2,
				}
				cl.Draw(screen, x, y, x+kbXStepValue, y+kbYStepValue)

				letterCode++
				if letterCode == uint8('Z'+1) {
					letterCode = uint8('<')
				}
			}
		} else {
			ebitenutil.DrawRect(screen, 0, y, float64(awg.GetWidth()), 30, color.RGBA{255, 255, 255, 255})
			ebitenutil.DrawRect(screen, 4, y+4, float64(awg.GetWidth())-8, 22, color.RGBA{55, 55, 55, 255})
			ebitenutil.DebugPrintAt(screen, "E N T E R", awg.GetWidth()/2-30, int(y+2))
		}
		rowCount++
	}
}

func (awg *AntiwordGame) GameOver() {
	fmt.Println("Game over!  Score: ", len(awg.playtrix))
}

func (awg *AntiwordGame) WordContainsLetter(letter uint8) bool {
	return awg.wordMap[letter] != nil
}

func (awg *AntiwordGame) ClaimLetterAtPosition(letter uint8, position int) (retval bool) {
	for i := range awg.wordMap[letter] {
		if awg.wordMap[letter][i].pos == position {
			awg.wordMap[letter][i].claim = true
			return true
		}
	}
	return
}

func (awg *AntiwordGame) BindALetter(letter uint8) bool {
	for i := range awg.wordMap[letter] {
		if !awg.wordMap[letter][i].claim {
			if !awg.wordMap[letter][i].bound {
				awg.wordMap[letter][i].bound = true
				return true
			}
		}
	}
	return false
}

func (awg *AntiwordGame) ResetBindings() {
	for letter := range awg.wordMap {
		for i := range awg.wordMap[letter] {
			awg.wordMap[letter][i].bound = false
		}
	}
}

func (awg *AntiwordGame) EnterPressed() {
	enteredWord := ""
	for i := range awg.playtrix[awg.entryWord] {
		if awg.playtrix[awg.entryWord][i].letter == 0 {
			return
		}
		enteredWord = enteredWord + fmt.Sprintf("%c", awg.playtrix[awg.entryWord][i].letter)
	}

	if enteredWord == awg.word {
		awg.GameOver()
		return
	}

	if !awg.dictMap[enteredWord] {
		fmt.Println("Word", enteredWord, "not found in dictionary.")
		return
	}

	delete(awg.dictMap, enteredWord)

	//Traverse the guessed word, letter by letter.
	skipList := make(map[int]bool)
	for i := range enteredWord {
		//First, match up "green" matches.
		if awg.WordContainsLetter(enteredWord[i]) {
			//The letter appears in The Word.
			if awg.ClaimLetterAtPosition(enteredWord[i], i) {
				awg.playtrix[awg.entryWord][i].state = RIGHTLETTERRIGHTPLACE
				awg.keys[enteredWord[i]] = RIGHTLETTERRIGHTPLACE
				skipList[i] = true
				//Enforcement map tracks the legal inputs for column i.
				for k := range awg.enforcementMap[i] {
					awg.enforcementMap[i][k] = false
				}
				awg.enforcementMap[i][enteredWord[i]] = true

			}
		}
	}

	awg.ResetBindings()
	for i := range enteredWord {
		if skipList[i] {
			//Don't bind perfect matches.
			continue
		}
		if awg.BindALetter(enteredWord[i]) {
			awg.playtrix[awg.entryWord][i].state = RIGHTLETTERWRONGPLACE
			if awg.keys[enteredWord[i]] == UNKNOWN {
				awg.keys[enteredWord[i]] = RIGHTLETTERWRONGPLACE
			}
			awg.enforcementMap[i][enteredWord[i]] = false
		} else {
			awg.playtrix[awg.entryWord][i].state = WRONGLETTER
			if awg.keys[enteredWord[i]] == UNKNOWN {
				awg.keys[enteredWord[i]] = WRONGLETTER
			}
			awg.enforcementMap[i][enteredWord[i]] = false
		}
	}

	dictWordsToDelete := make(map[string]bool)
	for k := range awg.dictMap {
		for _, letterA := range k {
			if awg.wordMap[uint8(letterA)] == nil {
				for _, letterB := range enteredWord {
					if letterA == letterB {
						dictWordsToDelete[k] = true
					}
				}
			}
		}
	}
	for k := range dictWordsToDelete {
		delete(awg.dictMap, k)
	}

	awg.playtrix = append(awg.playtrix, make([]ColorfulLetter, awg.GetWordSize(), awg.GetWordSize()))
	awg.entryWord++
	awg.entryLetter = 0

	/*
		fmt.Println(awg.word, " -- Enforcement map:")
		for i := range awg.enforcementMap {
			fmt.Print(i, ":")
			for k, v := range awg.enforcementMap[i] {
				if v {
					fmt.Print(fmt.Sprintf("%c", k), ",")

				}
			}
			fmt.Println()
		}
	*/

}

func (awg *AntiwordGame) Update(screen *ebiten.Image) error {

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {

		for awg.entryWord >= len(awg.playtrix) {
			awg.playtrix = append(awg.playtrix, make([]ColorfulLetter, awg.GetWordSize(), awg.GetWordSize()))
		}

		kbXStepValue := float64(awg.GetWidth() / 9)
		kbYStepValue := float64(awg.GetHeight()) * awg.GetKeyBoardStartRatio() / 4.0
		kbYStartValue := float64(awg.GetHeight()) - float64(awg.GetHeight())*awg.GetKeyBoardStartRatio()

		x, y := ebiten.CursorPosition()

		rowOffset := int((float64(y)-kbYStartValue)/kbYStepValue) * 9

		if rowOffset >= 27 {
			awg.EnterPressed()
		} else {
			columnOffset := int(float64(x) / kbXStepValue)
			enteredLetter := uint8('A') + uint8(rowOffset+columnOffset)
			if enteredLetter == uint8('Z')+1 {
				if awg.entryLetter > 0 {
					awg.entryLetter--
				}
				awg.playtrix[awg.entryWord][awg.entryLetter].letter = 0
			} else if awg.entryLetter < 5 &&
				awg.keys[enteredLetter] != WRONGLETTER &&
				awg.enforcementMap[awg.entryLetter][enteredLetter] {
				awg.playtrix[awg.entryWord][awg.entryLetter].letter = enteredLetter
				awg.entryLetter++
			}
		}
	}
	return nil
}

func main() {
	awg := &AntiwordGame{dict: dictionary.Dictionary}
	//awg.Reset([]string{})
	awg.Reset(nil)
	ebiten.SetWindowSize(awg.GetWidth(), awg.GetHeight())
	ebiten.SetWindowTitle("Antiword")
	if err := ebiten.RunGame(awg); err != nil {
		log.Fatal(err)
	}
}
