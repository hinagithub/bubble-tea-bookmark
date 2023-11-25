package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	cursor "github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/list"
	input "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const listHeight = 14
const defaultWidth = 20

var (
	// リストのスタイル
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			PaddingTop(1).
			PaddingBottom(1).
			PaddingLeft(2).
			Width(26)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)

	// テキストインプットのスタイル
	inputFocusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205")) // フォーカスしている文字の色
	inputBlurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // フォーカス外の文字の色
	inputCursorStyle         = inputFocusedStyle.Copy()                              // カーソルが当たっている文字の色
	inputNoStyle             = lipgloss.NewStyle()                                   //デフォルトの文字の色
	inputHelpStyle           = inputBlurredStyle.Copy()                              // ヘルプメッセージの文字の色
	inputCursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")) // カーソルモードヘルプの文字の色

	inputFocusedButton = inputFocusedStyle.Copy().Render("[ Submit ]")             //フォーカスしているボタンの色
	inputBlurredButton = fmt.Sprintf("[ %s ]", inputBlurredStyle.Render("Submit")) //フォーカス外のボタンの色
)

type item string

func (i item) FilterValue() string { return "" }

type Favorite struct {
	Title string `json:"title"`
	Url   string `json:"url"`
}

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list       list.Model
	favorites  []Favorite
	url        string
	choice     string
	quitting   bool
	mode       string
	inputs     []input.Model
	focusIndex int
	cursorMode cursor.Mode
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// タイトル追加モード
	if m.mode == "addTitle" {
		return m.UpdateInputs(msg)
	}

	// 一覧モード
	if m.mode == "list" {
		return m.UpdateList(msg)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// 一覧モードUpdate
func (m model) UpdateList(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		// 起動時にウィンドウサイズを設定
		m.list.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		// キャンセル
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		// エンター
		case "enter":
			item, ok := m.list.SelectedItem().(item)
			for _, f := range m.favorites {
				if f.Title == string(item) {
					m.url = f.Url
				}
			}
			if ok {
				m.choice = string(item)
				exec.Command("open", m.url).Start()
			}
			return m, tea.Quit

		// 追加モード
		case "i":
			m.mode = "addTitle"
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

// 追加モードUpdate
func (m model) UpdateInputs(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		// Change cursor mode
		case "ctrl+r":
			m.cursorMode++
			if m.cursorMode > cursor.CursorHide {
				m.cursorMode = cursor.CursorBlink
			}
			cmds := make([]tea.Cmd, len(m.inputs))
			for i := range m.inputs {
				cmds[i] = m.inputs[i].Cursor.SetMode(m.cursorMode)
			}
			return m, tea.Batch(cmds...)

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// submitボタン
			if s == "enter" && m.focusIndex == len(m.inputs) {

				// リストを更新
				m.favorites = append(m.favorites, Favorite{
					Title: m.inputs[0].Value(),
					Url:   m.inputs[1].Value(),
				})

				// リストをJSONにエンコード
				jsonData, err := json.MarshalIndent(m.favorites, "", "    ")
				if err != nil {
					log.Fatal(err)
				}

				// ファイルに書き込む
				err = ioutil.WriteFile("favorites.json", jsonData, os.ModePerm)
				if err != nil {
					log.Fatal(err)
				}
				return m, tea.Quit
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					// Set focused state
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = inputFocusedStyle
					m.inputs[i].TextStyle = inputFocusedStyle
					continue
				}
				// Remove focused state
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = inputNoStyle
				m.inputs[i].TextStyle = inputNoStyle
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Handle character input and blinking
	cmd := m.updateInputs(msg)

	return m, cmd
	// switch msg := msg.(type) {
	// case tea.KeyMsg:
	// 	switch msg.String() {
	// 	case "ctrl+q":
	// 		m.mode = "list"
	// 		m.inputs[0].Reset()
	// 		return m, nil
	// 	case "enter":

	// 		// 空文字ならリストに戻る
	// 		if m.inputs[0].Value() == "" {
	// 			m.mode = "list"
	// 			return m, nil
	// 		}

	// 		// リストを更新
	// 		m.favorites = append(m.favorites, Favorite{
	// 			Title: m.inputs[0].Value(),
	// 			Url:   "URL",
	// 		})
	// 		// リストをJSONにエンコード
	// 		jsonData, err := json.MarshalIndent(m.favorites, "", "    ")
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}
	// 		// ファイルに書き込む
	// 		err = ioutil.WriteFile("favorites.json", jsonData, os.ModePerm)
	// 		if err != nil {
	// 			log.Fatal(err)
	// 		}

	// 		// list.Add的な項目追加の関数はないためNewで再生成
	// 		var items []list.Item
	// 		for _, f := range m.favorites {
	// 			items = append(items, item(f.Title))
	// 		}
	// 		m.list = list.New(items, itemDelegate{}, defaultWidth, listHeight)
	// 		m.mode = "list"
	// 		m.inputs[0].Reset()
	// 		return m, nil
	// 	}

	// }
	// var cmd tea.Cmd
	// // m.inputs, cmd = m.inputs[0].Update(msg)
	// return m, cmd
}

func (m *model) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

/*
一覧モードView
*/
func (m model) View() string {
	if m.mode == "addTitle" {
		return m.addingTaskView()
	}
	if m.choice != "" {
		return quitTextStyle.Render(fmt.Sprintf("%s(%s) を選択", m.choice, m.url))
	}
	if m.quitting {
		return quitTextStyle.Render(fmt.Sprintf("キャンセルしました! %v", m.favorites))
	}
	return "\n" + m.list.View()
}

// 追加モードView
func (m model) addingTaskView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("🌷 My Favorite Links"))
	b.WriteString("\n")
	b.WriteString("\n")
	b.WriteString("Type Title & URL.")
	b.WriteString("\n")
	b.WriteString("\n")

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &inputBlurredButton
	if m.focusIndex == len(m.inputs) {
		button = &inputFocusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	b.WriteString(inputHelpStyle.Render("cursor mode is "))
	b.WriteString(inputCursorModeHelpStyle.Render(m.cursorMode.String()))
	b.WriteString(inputHelpStyle.Render(" (ctrl+r to change style)"))

	return b.String()
}

func main() {

	// お気に入りリストの読み込み
	raw, err := ioutil.ReadFile("favorites.json")
	if err != nil {
		panic(err)
	}
	var favorites []Favorite
	json.Unmarshal(raw, &favorites)

	var items []list.Item
	for _, f := range favorites {
		items = append(items, item(f.Title))
	}

	// 一覧モデルの設定
	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "🌷 My Favorite Links"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	// テキストインプットモデルの設定
	inputs := make([]input.Model, 2)

	for i := range inputs {
		t := input.New()
		t.Cursor.Style = inputCursorStyle

		switch i {
		case 0:
			t.Placeholder = "Title"
			t.Focus()
			t.PromptStyle = inputFocusedStyle
			t.TextStyle = inputFocusedStyle
			t.CharLimit = 30

		case 1:
			t.Placeholder = "URL"
			t.CharLimit = 256
		}
		inputs[i] = t
	}

	m := model{
		list:      l,
		favorites: favorites,
		inputs:    inputs,
	}
	m.mode = "list"

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
