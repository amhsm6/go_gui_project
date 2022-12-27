package main

import (
    "github.com/gotk3/gotk3/gtk"
    "log"
    "strings"
    "os"
    "io"
    "path/filepath"
    "fmt"
)

type Menu struct {
    entries []MenuEntry
}

func (m *Menu) AddEntryWithAction(label string, next *Menu, action func(), entryType string, marginTop int) {
    if next == nil { next = m }

    m.entries = append(m.entries, MenuEntry{
        Label: label,
        Next: next,
        Action: action,
        Type: entryType,
        MarginTop: marginTop,
        InputBuffer: nil,
    })
}

func (m *Menu) AddEntry(label string, next *Menu, entryType string, marginTop int) {
    m.AddEntryWithAction(label, next, nil, entryType, marginTop)
}

type MenuEntry struct {
    Label string
    Next *Menu
    Action func()
    Type string
    MarginTop int
    InputBuffer *gtk.EntryBuffer
}

func (e MenuEntry) Use() *Menu {
    if e.Action != nil { e.Action() }

    return e.Next
}

func (m Menu) ProcessNextMenu(box *gtk.Box) {
    box.GetChildren().Foreach(func (child any) {
        btn, _ := child.(*gtk.Widget)
        btn.Destroy()
    })

    for i, entry := range m.entries {
        if entry.Type == "button" {
            btn, err := gtk.ButtonNewWithLabel(entry.Label)

            if err != nil {
                log.Panic(err)
            }

            btn.SetMarginTop(entry.MarginTop)

            currentEntry := entry
            btn.Connect("clicked", func() {
                currentEntry.Use().ProcessNextMenu(box)
            })

            box.Add(btn)
        } else if entry.Type == "label" {
            entry.Label = strings.Replace(entry.Label, "$", templatesRoot, 1)
            entry.Label = strings.Replace(entry.Label, "#", projectPath, 1)

            label, err := gtk.LabelNew(entry.Label)

            if err != nil {
                log.Panic(err)
            }

            label.SetMarginTop(entry.MarginTop)

            box.Add(label)
        } else if entry.Type == "input" {
            input, err := gtk.EntryNew()

            if err != nil {
                log.Panic(err)
            }

            input.SetPlaceholderText(entry.Label)
            input.SetMarginTop(entry.MarginTop)

            buf, err := input.GetBuffer()

            if err != nil {
                log.Panic(err)
            }

            m.entries[i].InputBuffer = buf

            box.Add(input)
        } else {
            log.Panic()
        }
    }

    box.ShowAll()
}

func (m Menu) GtkWidget() *gtk.Widget {
    box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)

    if err != nil {
        log.Panic(err)
    }

    m.ProcessNextMenu(box)

    return &box.Widget
}

var projectPath, templatesRoot string

func makeMainMenu(win *gtk.Window) *Menu {
    var mainMenu, optionsMenu Menu

    mainMenu.AddEntry("Приложение для инициализации проектов по шаблонам", nil, "label", 5)

    mainMenu.AddEntry("Расположение проекта: #", nil, "label", 24)
    mainMenu.AddEntryWithAction("Изменить...", nil, func() {
        fileChooserDialog, err := gtk.FileChooserDialogNewWith1Button(
            "Расположение проекта",
            win,
            gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER,
            "Выбрать",
            gtk.RESPONSE_YES,
        )

        if err != nil {
            log.Panic(err)
        }

        fileChooserDialog.Run()

        projectPath = fileChooserDialog.GetFilename()

        fileChooserDialog.Destroy()
    }, "button", 5)

    mainMenu.AddEntry("Название шаблона", nil, "input", 24)

    mainMenu.AddEntryWithAction("Инициализировать проект", nil, func() {
        templateName, err := mainMenu.entries[3].InputBuffer.GetText()

        if err != nil {
            log.Panic(err)
        }

        templatePath := filepath.Join(templatesRoot, templateName)

        confirmWin, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)

        if err != nil {
            log.Panic(err)
        }

        box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)

        if err != nil {
            log.Panic(err)
        }

        box.SetMarginTop(24)
        box.SetMarginBottom(24)
        box.SetMarginStart(24)
        box.SetMarginEnd(24)

        if templateName == "" {
            label, err := gtk.LabelNew("Ошибка: не указано название шаблона")

            if err != nil {
                log.Panic(err)
            }

            box.Add(label)
        } else if !Exists(templatePath) {
            label, err := gtk.LabelNew(fmt.Sprintf("Ошибка: шаблона %s не существует", templateName))

            if err != nil {
                log.Panic(err)
            }

            box.Add(label)
        } else {
            displayedProjectPath := projectPath

            if displayedProjectPath == "" {
                displayedProjectPath = "текущую дерикторию"
            }

            label, err := gtk.LabelNew(fmt.Sprintf("Копирование файлов\nиз %s\nв %s", templatePath, displayedProjectPath))

            if err != nil {
                log.Panic(err)
            }

            box.Add(label)

            button, err := gtk.ButtonNewWithLabel("Подтвердить")

            if err != nil {
                log.Panic(err)
            }

            button.Connect("clicked", func() {
                box.GetChildren().Foreach(func(child any) {
                    widget, ok := child.(*gtk.Widget)

                    if !ok {
                        return
                    }

                    widget.Destroy()
                })

                label, err := gtk.LabelNew("Копирование...")

                if err != nil {
                    log.Panic(err)
                }

                box.Add(label)

                box.ShowAll()

                err = CopyDirectory(templatePath, projectPath)

                if err != nil {
                    label.SetLabel("Ошибка:\n" + err.Error())
                } else {
                    label.SetLabel("Инициализация проекта прошла успешно")
                }
            })

            box.Add(button)
        }

        confirmWin.Add(box)

        confirmWin.ShowAll()
    }, "button", 30)
    mainMenu.AddEntry("Настройки", &optionsMenu, "button", 10)

    optionsMenu.AddEntry("Путь до папки с шаблонами: $", nil, "label", 10)
    optionsMenu.AddEntryWithAction("Изменить...", nil, func() {
        fileChooserDialog, err := gtk.FileChooserDialogNewWith1Button(
            "Папка с шаблонами",
            win,
            gtk.FILE_CHOOSER_ACTION_SELECT_FOLDER,
            "Выбрать",
            gtk.RESPONSE_YES,
        )

        if err != nil {
            log.Panic(err)
        }

        fileChooserDialog.Run()

        templatesRoot = fileChooserDialog.GetFilename()

        f, err := os.OpenFile("config", os.O_CREATE | os.O_WRONLY | os.O_TRUNC, 644)
        if err != nil {
            log.Panic(err)
        }

        f.WriteString(templatesRoot)

        f.Close()

        fileChooserDialog.Destroy()
    }, "button", 5)
    optionsMenu.AddEntry("Назад", &mainMenu, "button", 10)

    return &mainMenu
}

func CopyDirectory(src, dest string) error {
    entries, err := os.ReadDir(src)

    if err != nil {
        return err
    }

    for _, entry := range entries {
        sourcePath := filepath.Join(src, entry.Name())
        destPath := filepath.Join(dest, entry.Name())

        fileInfo, err := os.Stat(sourcePath)
        if err != nil {
            return err
        }

        switch fileInfo.Mode() & os.ModeType {
        case os.ModeDir:
            if err := CreateIfNotExists(destPath, 0755); err != nil {
                return err
            }
            if err := CopyDirectory(sourcePath, destPath); err != nil {
                return err
            }

        default:
            if err := Copy(sourcePath, destPath); err != nil {
                return err
            }
        }
    }
    return nil
}

func Copy(srcFile, dstFile string) error {
    out, err := os.Create(dstFile)
    if err != nil {
        return err
    }

    defer out.Close()

    in, err := os.Open(srcFile)
    defer in.Close()
    if err != nil {
        return err
    }

    _, err = io.Copy(out, in)
    if err != nil {
        return err
    }

    return nil
}

func Exists(filePath string) bool {
    if _, err := os.Stat(filePath); os.IsNotExist(err) {
        return false
    }

    return true
}

func CreateIfNotExists(dir string, perm os.FileMode) error {
    if Exists(dir) {
        return nil
    }

    if err := os.MkdirAll(dir, perm); err != nil {
        return fmt.Errorf("Failed to create directory: '%s', error: '%s'", dir, err.Error())
    }

    return nil
}

func main() {
    bytes, err := os.ReadFile("config")
    if err == nil {
        templatesRoot = string(bytes)
    }

    gtk.Init(nil)

    win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)

    if err != nil {
        log.Panic(err)
    }

    win.Connect("destroy", func() {
        gtk.MainQuit()
    })

    box, err := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 12)

    if err != nil {
        log.Panic(err)
    }

    box.SetMarginTop(24)
    box.SetMarginBottom(24)
    box.SetMarginStart(24)
    box.SetMarginEnd(24)

    win.Add(box)

    box.Add(makeMainMenu(win).GtkWidget())

    win.ShowAll()

    gtk.Main()
}
