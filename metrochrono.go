package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ChronoData represents the data we need to save/load for each chronometer
type ChronoData struct {
	ID           int           `json:"id"`
	DisplayLabel string        `json:"displayLabel"`
	ElapsedTime  time.Duration `json:"elapsedTime"`
	IsRunning    bool          `json:"isRunning"`
}

// SaveData represents all chronometers for saving/loading
type SaveData struct {
	Chronometers []ChronoData `json:"chronometers"`
	SaveTime     time.Time    `json:"saveTime"`
}

type Chronometer struct {
	startTime    time.Time
	elapsedTime  time.Duration
	isRunning    bool
	displayLabel string
	id           int
}

func NewChronometer(id int) *Chronometer {
	return &Chronometer{
		elapsedTime:  0,
		isRunning:    false,
		displayLabel: fmt.Sprintf("Timer %d", id),
		id:           id,
	}
}

func (c *Chronometer) Start() {
	if !c.isRunning {
		c.startTime = time.Now().Add(-c.elapsedTime)
		c.isRunning = true
	}
}

func (c *Chronometer) Stop() {
	if c.isRunning {
		c.elapsedTime = time.Since(c.startTime)
		c.isRunning = false
	}
}

func (c *Chronometer) Reset() {
	c.elapsedTime = 0
	if c.isRunning {
		c.startTime = time.Now()
	}
}

func (c *Chronometer) GetElapsedTime() time.Duration {
	if c.isRunning {
		return time.Since(c.startTime)
	}
	return c.elapsedTime
}

func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	milliseconds := int(d.Milliseconds()) % 1000

	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, milliseconds)
}

func parseDuration(s string) (time.Duration, error) {
	// Split by : and .
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time format")
	}

	secParts := strings.Split(parts[2], ".")
	if len(secParts) != 2 {
		return 0, fmt.Errorf("invalid seconds format")
	}

	// Parse each part
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}

	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.Atoi(secParts[0])
	if err != nil {
		return 0, err
	}

	millis, err := strconv.Atoi(secParts[1])
	if err != nil {
		return 0, err
	}

	// Calculate total duration
	duration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(millis)*time.Millisecond

	return duration, nil
}

type ChronoManager struct {
	chronometers []*Chronometer
	mutex        sync.Mutex
}

func NewChronoManager(count int) *ChronoManager {
	cm := &ChronoManager{
		chronometers: make([]*Chronometer, count),
	}
	for i := 0; i < count; i++ {
		cm.chronometers[i] = NewChronometer(i + 1)
	}
	return cm
}

func (cm *ChronoManager) StartChronometer(id int) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Stop all running chronometers
	for _, c := range cm.chronometers {
		if c.isRunning {
			c.Stop()
		}
	}

	// Start the selected chronometer
	if id >= 0 && id < len(cm.chronometers) {
		cm.chronometers[id].Start()
	}
}

func (cm *ChronoManager) SaveToFile(filename string) error {
	data := SaveData{
		Chronometers: make([]ChronoData, len(cm.chronometers)),
		SaveTime:     time.Now(),
	}

	for i, c := range cm.chronometers {
		data.Chronometers[i] = ChronoData{
			ID:           c.id,
			DisplayLabel: c.displayLabel,
			ElapsedTime:  c.GetElapsedTime(),
			IsRunning:    c.isRunning,
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, jsonData, 0644)
}

func (cm *ChronoManager) LoadFromFile(filename string) error {
	jsonData, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var data SaveData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return err
	}

	// Stop all running chronometers first
	for _, c := range cm.chronometers {
		c.Stop()
	}

	// Update chronometer states
	for _, cd := range data.Chronometers {
		// Find the corresponding chronometer by ID
		for i, c := range cm.chronometers {
			if c.id == cd.ID {
				cm.chronometers[i].displayLabel = cd.DisplayLabel
				cm.chronometers[i].elapsedTime = cd.ElapsedTime
				// If it was running, start it again
				if cd.IsRunning {
					cm.chronometers[i].Start()
				}
				break
			}
		}
	}

	return nil
}

func (cm *ChronoManager) SaveToCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"Timer ID", "Label", "Elapsed Time"}); err != nil {
		return err
	}

	// Write data
	for _, c := range cm.chronometers {
		elapsed := formatDuration(c.GetElapsedTime())
		if err := writer.Write([]string{
			fmt.Sprintf("%d", c.id),
			c.displayLabel,
			elapsed,
		}); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	app := tview.NewApplication()

	// Create chronometer manager with 15 chronometers
	manager := NewChronoManager(15)

	// Main layout grid
	grid := tview.NewGrid().
		SetRows(0, 3). // Main area for chronometers, 3 rows for buttons
		SetColumns(0)

	// Create a grid for chronometers: 3 columns, 5 rows
	chronoGrid := tview.NewGrid().
		SetRows(0, 0, 0, 0, 0).
		SetColumns(0, 0, 0)

	chronometersUI := make([]*tview.Flex, 15)
	statusTexts := make([]*tview.TextView, 15)
	labelInputs := make([]*tview.InputField, 15)

	// Create UI for each chronometer
	for i := 0; i < 15; i++ {
		chron := manager.chronometers[i]
		chronUI := tview.NewFlex().SetDirection(tview.FlexRow)

		// Label input for this chronometer
		labelInput := tview.NewInputField().
			SetLabel("Label: ").
			SetText(chron.displayLabel).
			SetFieldWidth(80).
			SetDoneFunc(func(key tcell.Key) {
				// This will be set properly below
			})

		// Store for later reference
		labelInputs[i] = labelInput

		// Timer display
		timeText := tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetDynamicColors(true).
			SetText("[yellow]00:00:00.000")

		// Timer buttons
		buttonFlex := tview.NewFlex().SetDirection(tview.FlexColumn)

		// Get the ID for button callbacks
		id := i // Important: Create a new variable to capture the current value of i

		startButton := tview.NewButton("Start").SetSelectedFunc(func() {
			manager.StartChronometer(id)
		}).SetLabelColor(tcell.ColorGreen)

		stopButton := tview.NewButton("Stop").SetSelectedFunc(func() {
			manager.chronometers[id].Stop()
		})

		resetButton := tview.NewButton("Reset").SetSelectedFunc(func() {
			manager.chronometers[id].Reset()
		})

		startButton.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
			if action == tview.MouseLeftClick {
				manager.StartChronometer(id)
			}
			return action, event
		})

		stopButton.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
			if action == tview.MouseLeftClick {
				manager.chronometers[id].Stop()
			}
			return action, event
		})

		resetButton.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
			if action == tview.MouseLeftClick {
				manager.chronometers[id].Reset()
			}
			return action, event
		})

		buttonFlex.AddItem(startButton, 0, 1, false)
		buttonFlex.AddItem(stopButton, 0, 1, false)
		buttonFlex.AddItem(resetButton, 0, 1, false)

		// Status text
		statusText := tview.NewTextView().
			SetTextAlign(tview.AlignCenter).
			SetText("Status: Stopped")

		statusTexts[i] = statusText

		// Add components to chronometer UI
		chronUI.AddItem(labelInput, 3, 0, true).
			AddItem(timeText, 3, 0, false).
			AddItem(buttonFlex, 3, 0, false).
			AddItem(statusText, 1, 0, false)

		chronUI.SetBorder(true).SetTitle(fmt.Sprintf(" Timer %d ", i+1))
		chronometersUI[i] = chronUI

		// Add to the grid - calculate row and column
		col := i % 3
		row := i / 3
		chronoGrid.AddItem(chronUI, row, col, 1, 1, 0, 0, false)
	}

	// Now that we have all the input fields, set their proper DoneFunc
	for i, labelInput := range labelInputs {
		// Create a closure with the correct id
		id := i
		labelInput.SetDoneFunc(func(key tcell.Key) {
			manager.chronometers[id].displayLabel = labelInput.GetText()
		})
	}

	// Button panel at the bottom
	buttonPanel := tview.NewFlex().SetDirection(tview.FlexColumn)

	// Save button
	saveButton := tview.NewButton("Save").SetSelectedFunc(func() {
		form := tview.NewForm()
		form.AddInputField("Filename", "timers.json", 20, nil, nil)
		form.AddButton("Save", func() {
			filename := form.GetFormItem(0).(*tview.InputField).GetText()
			err := manager.SaveToFile(filename)
			var modalText string
			if err != nil {
				modalText = fmt.Sprintf("Error saving: %v", err)
			} else {
				modalText = fmt.Sprintf("Successfully saved to %s", filename)
			}

			modal := tview.NewModal().
				SetText(modalText).
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					app.SetRoot(grid, true)
				})
			app.SetRoot(modal, false)
		})
		form.AddButton("Cancel", func() {
			app.SetRoot(grid, true)
		})
		form.SetBorder(true).SetTitle("Save Timers")
		form.SetCancelFunc(func() {
			app.SetRoot(grid, true)
		})
		app.SetRoot(form, true)
	})

	// Load button
	loadButton := tview.NewButton("Load").SetSelectedFunc(func() {
		form := tview.NewForm()
		form.AddInputField("Filename", "timers.json", 20, nil, nil)
		form.AddButton("Load", func() {
			filename := form.GetFormItem(0).(*tview.InputField).GetText()
			err := manager.LoadFromFile(filename)
			var modalText string
			if err != nil {
				modalText = fmt.Sprintf("Error loading: %v", err)
			} else {
				modalText = fmt.Sprintf("Successfully loaded from %s", filename)
				// Update the UI with the loaded values
				for i, c := range manager.chronometers {
					labelInputs[i].SetText(c.displayLabel)
				}
			}

			modal := tview.NewModal().
				SetText(modalText).
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					app.SetRoot(grid, true)
				})
			app.SetRoot(modal, false)
		})
		form.AddButton("Cancel", func() {
			app.SetRoot(grid, true)
		})
		form.SetBorder(true).SetTitle("Load Timers")
		form.SetCancelFunc(func() {
			app.SetRoot(grid, true)
		})
		app.SetRoot(form, true)
	})

	// Export CSV button
	exportButton := tview.NewButton("Export CSV").SetSelectedFunc(func() {
		form := tview.NewForm()
		form.AddInputField("Filename", "timers.csv", 20, nil, nil)
		form.AddButton("Export", func() {
			filename := form.GetFormItem(0).(*tview.InputField).GetText()
			err := manager.SaveToCSV(filename)
			var modalText string
			if err != nil {
				modalText = fmt.Sprintf("Error exporting: %v", err)
			} else {
				modalText = fmt.Sprintf("Successfully exported to %s", filename)
			}

			modal := tview.NewModal().
				SetText(modalText).
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					app.SetRoot(grid, true)
				})
			app.SetRoot(modal, false)
		})
		form.AddButton("Cancel", func() {
			app.SetRoot(grid, true)
		})
		form.SetBorder(true).SetTitle("Export to CSV")
		form.SetCancelFunc(func() {
			app.SetRoot(grid, true)
		})
		app.SetRoot(form, true)
	})

	// Quit button
	quitButton := tview.NewButton("Quit").SetSelectedFunc(func() {
		modal := tview.NewModal().
			SetText("Are you sure you want to quit?").
			AddButtons([]string{"Quit", "Cancel"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				if buttonLabel == "Quit" {
					app.Stop()
				} else {
					app.SetRoot(grid, true)
				}
			})
		app.SetRoot(modal, false)
	})

	buttonPanel.AddItem(saveButton, 0, 1, false)
	buttonPanel.AddItem(loadButton, 0, 1, false)
	buttonPanel.AddItem(exportButton, 0, 1, false)
	buttonPanel.AddItem(quitButton, 0, 1, false)

	// Add chronometers and button panel to main grid
	grid.AddItem(chronoGrid, 0, 0, 1, 1, 0, 0, true)
	grid.AddItem(buttonPanel, 1, 0, 1, 1, 0, 0, false)

	// Update the timer displays every 10 milliseconds
	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			app.QueueUpdateDraw(func() {
				for i, c := range manager.chronometers {
					chronUI := chronometersUI[i]
					timeText := chronUI.GetItem(1).(*tview.TextView)
					statusText := statusTexts[i]

					elapsed := c.GetElapsedTime()
					timeText.SetText(fmt.Sprintf("[yellow]%s", formatDuration(elapsed)))

					if c.isRunning {
						statusText.SetText("Status: Running")
						chronUI.SetTitle(fmt.Sprintf(" Timer %d [green]● ", i+1))
					} else {
						statusText.SetText("Status: Stopped")
						chronUI.SetTitle(fmt.Sprintf(" Timer %d ", i+1))
					}
				}
			})
		}
	}()

	// Handle keyboard shortcuts
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			app.Stop()
			return nil
		}
		return event
	})

	// Enable mouse support
	app.EnableMouse(true)

	// Run the application
	if err := app.SetRoot(grid, true).Run(); err != nil {
		panic(err)
	}
}
