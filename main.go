package main

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"

	"github.com/energye/systray"
	"github.com/taodev/pkg/config"
)

var (
	appName      = "gotray"
	exePath      string
	dirPath      string
	configPath   string
	globalConfig Config
	menuConfig   MenuConfig
)

//go:embed icon/vscode.ico
var vscodeIcon []byte

func main() {
	exePath, _ = os.Executable()
	dirPath = filepath.Dir(exePath)
	f, err := os.OpenFile(filepath.Join(dirPath, "app.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	configPath = filepath.Join(dirPath, "config.yaml")
	if err := config.LoadYAML(configPath, &globalConfig); err != nil {
		globalConfig.Path = "./menu.yaml"
		if err := config.SaveYAML(configPath, &globalConfig); err != nil {
			slog.Error("save config failed", "err", err)
			Alert("保存配置失败", fmt.Sprintf("path: %s, err: %s", configPath, err.Error()), 0)
			return
		}
	}

	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(vscodeIcon)
	systray.SetTitle("gotray")
	systray.SetTooltip("gotray")
	systray.SetOnClick(func(menu systray.IMenu) {
		menu.ShowMenu()
	})
	systray.SetOnRClick(func(menu systray.IMenu) {
		menu.ShowMenu()
	})

	initMenu()
	initSystemMenu()
}

func onExit() {

}

func initSystemMenu() {
	systray.AddSeparator()

	refreshMenu := systray.AddMenuItem("刷新", "")
	refreshMenu.Click(func() {
		// 重启程序
		cmd := exec.Command(exePath, os.Args[1:]...)
		if err := cmd.Start(); err != nil {
			slog.Error("start cmd failed", "cmd", cmd, "err", err)
			Alert("启动失败", err.Error(), 0)
		}

		systray.Quit()
	})

	autoStartMenu := systray.AddMenuItemCheckbox("开机启动", "", checkAutoStart())
	autoStartMenu.Click(func() {
		if autoStartMenu.Checked() {
			disableAutoStart()
		} else {
			enableAutoStart()
		}

		if checkAutoStart() {
			autoStartMenu.Check()
		} else {
			autoStartMenu.Uncheck()
		}
	})

	quitMenu := systray.AddMenuItem("退出", "")
	quitMenu.Click(func() {
		systray.Quit()
	})
}

func initMenu() {
	menuPath := globalConfig.Path
	if !filepath.IsAbs(menuPath) {
		menuPath = filepath.Join(dirPath, menuPath)
	}
	if err := config.LoadYAML(menuPath, &menuConfig); err != nil {
		if err := config.SaveYAML(menuPath, &menuConfig); err != nil {
			slog.Error("save config failed", "path", menuPath, "err", err)
			Alert("保存菜单失败", fmt.Sprintf("path: %s, err: %s", menuPath, err.Error()), 0)
			return
		}
	}

	for _, item := range menuConfig.Menu {
		addMenu(nil, &item)
	}
}

func addMenu(parent *systray.MenuItem, item *MenuItem) {
	var menu *systray.MenuItem
	if parent == nil {
		menu = systray.AddMenuItem(item.Title, item.Title)
	} else {
		menu = parent.AddSubMenuItem(item.Title, item.Title)
	}
	menu.SetIcon(vscodeIcon)

	if len(item.Items) > 0 {
		for _, subItem := range item.Items {
			addMenu(menu, &subItem)
		}
	} else {
		menu.Click(func() {
			if item.Cmd != nil {
				cmd := item.Cmd
				// dir := ""
				if cmdItem, ok := menuConfig.Cmds[item.Cmd[0]]; ok {
					// 合并 []string
					cmd = append(cmdItem.Cmd, item.Cmd[1:]...)
					// dir = cmdItem.Dir
				}
				slog.Info("run cmd", "cmd", cmd)
				app := exec.Command(cmd[0], cmd[1:]...)
				// app.Dir = dir
				app.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				if err := app.Start(); err != nil {
					slog.Error("start cmd failed", "cmd", item.Cmd, "err", err)
					Alert("启动失败", err.Error(), 0)
				}
			}
		})
	}
}

func enableAutoStart() {
	k, _, err := registry.CreateKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE)
	if err != nil {
		slog.Error("reg error", "err", err)
		Alert("开机启动设置失败", err.Error(), 0)
		return
	}
	defer k.Close()

	err = k.SetStringValue(appName, fmt.Sprintf(`"%s"`, exePath))

	if err != nil {
		slog.Error("set value error", "err", err)
		Alert("开机启动设置失败", err.Error(), 0)
		return
	}
}

func disableAutoStart() {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE)
	if err != nil {
		slog.Error("reg error", "err", err)
		Alert("开机启动设置失败", err.Error(), 0)
		return
	}
	defer k.Close()
	_ = k.DeleteValue(appName)
}

func checkAutoStart() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.QUERY_VALUE)
	if err != nil {
		slog.Error("reg error", "err", err)
		return false
	}
	defer k.Close()

	_, _, err = k.GetStringValue(appName)
	return err == nil
}

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	procMessageBox = user32.NewProc("MessageBoxW")
)

func Alert(title, text string, uType uint) int {
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		slog.Error("Alert utf16 error", "err", err)
	}
	textPtr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		slog.Error("Alert utf16 error", "err", err)
	}

	ret, _, _ := procMessageBox.Call(
		0,
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(uType),
	)
	return int(ret)
}
