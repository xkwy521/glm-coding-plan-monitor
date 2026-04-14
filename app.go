package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"fyne.io/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// --- Windows API: 隐藏任务栏图标 ---
var (
	modUser32         = syscall.NewLazyDLL("user32.dll")
	procFindWindow    = modUser32.NewProc("FindWindowW")
	procGetWindowLong = modUser32.NewProc("GetWindowLongW")
	procSetWindowLong = modUser32.NewProc("SetWindowLongW")
)

const (
	wsExToolWindow = 0x00000080
	wsExAppWindow  = 0x00040000
)

func (a *App) hideFromTaskbar() {
	title, _ := syscall.UTF16PtrFromString("GLM 使用量监控")
	hwnd, _, _ := procFindWindow.Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd == 0 {
		return
	}
	gwlExStyle := int32(-20) // GWL_EX_STYLE
	style, _, _ := procGetWindowLong.Call(hwnd, uintptr(gwlExStyle))
	procSetWindowLong.Call(hwnd, uintptr(gwlExStyle), (style &^ wsExAppWindow)|wsExToolWindow)
}

// AppConfig 应用配置结构体
type AppConfig struct {
	APIKey          string   `json:"api_key"`
	RefreshInterval int      `json:"refresh_interval"`
	LayoutMode      string   `json:"layout_mode"`
	DisplayItems    []string `json:"display_items"`
	WindowWidth     int      `json:"window_width"`
	GlassMode       bool     `json:"glass_mode"`
	AutoStart       bool     `json:"auto_start"`
	WindowMode      string   `json:"window_mode"` // "floating" 或 "tray"
}

// QuotaMonitorData API 数据传输结构体
type QuotaMonitorData struct {
	Level           string  `json:"level"`
	FiveHourUsedPct float64 `json:"five_hour_used_pct"`
	FiveHourLeftPct float64 `json:"five_hour_left_pct"`
	NextRefreshTime string  `json:"next_refresh_time"`
	WeeklyUsedPct   float64 `json:"weekly_used_pct"`
	WeeklyLeftPct   float64 `json:"weekly_left_pct"`
	McpCurrent      int     `json:"mcp_current"`
	McpTotal        int     `json:"mcp_total"`
	McpLeft         int     `json:"mcp_left"`
	McpUsagePct     float64 `json:"mcp_usage_pct"`
	LastError       string  `json:"last_error"`
}

type apiRawResponse struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
	Data    struct {
		Level  string         `json:"level"`
		Limits []apiLimitItem `json:"limits"`
	} `json:"data"`
}

type apiLimitItem struct {
	Type        string  `json:"type"`
	Percentage  float64 `json:"percentage"`
	Usage       int     `json:"usage"`
	Current     int     `json:"currentValue"`
	Remaining   int     `json:"remaining"`
	NextResetMs int64   `json:"nextResetTime"`
}

// App 应用结构体
type App struct {
	ctx          context.Context
	config       AppConfig
	cached       *QuotaMonitorData
	mu           sync.RWMutex
	ticker       *time.Ticker
	stopCh       chan struct{}
	systrayReady bool
	windowShown  bool
	mShow        *systray.MenuItem
	mRefresh     *systray.MenuItem
	mSettings    *systray.MenuItem
	mExit        *systray.MenuItem
}

func NewApp() *App {
	return &App{
		config: defaultConfig(),
		stopCh: make(chan struct{}),
	}
}

func defaultConfig() AppConfig {
	return AppConfig{
		APIKey:          "",
		RefreshInterval: 5,
		LayoutMode:      "single-line",
		DisplayItems:    []string{"5h_used", "next_refresh"},
		WindowWidth:     200,
		GlassMode:       true,
		WindowMode:      "floating",
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.loadConfig()
	a.startSystray()
	a.startPolling()

	go func() {
		time.Sleep(100 * time.Millisecond)
		a.hideFromTaskbar()
		a.windowShown = true
		runtime.WindowShow(a.ctx)
	}()
}

func (a *App) shutdown(ctx context.Context) {
	a.stopPolling()
	a.stopSystray()
}

// --- 配置管理 ---

func configPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
	}
	dir := filepath.Join(appData, "GLM-Monitor")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "config.json")
}

func (a *App) loadConfig() {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		a.config = defaultConfig()
		return
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		a.config = defaultConfig()
		return
	}
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = 5
	}
	if cfg.LayoutMode == "" {
		cfg.LayoutMode = "single-line"
	}
	if len(cfg.DisplayItems) == 0 {
		cfg.DisplayItems = []string{"5h_used", "next_refresh"}
	}
	if cfg.WindowWidth <= 0 {
		cfg.WindowWidth = 200
	}
	if cfg.WindowMode == "" || cfg.WindowMode == "taskbar" {
		cfg.WindowMode = "floating"
	}
	a.mu.Lock()
	a.config = cfg
	a.mu.Unlock()
}

func (a *App) writeConfig() error {
	data, err := json.MarshalIndent(a.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0644)
}

func (a *App) LoadConfig() AppConfig {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.config
}

func (a *App) SaveConfig(cfg AppConfig) error {
	a.mu.Lock()
	oldMode := a.config.WindowMode
	a.config = cfg
	a.mu.Unlock()

	if err := a.writeConfig(); err != nil {
		return err
	}

	// 模式切换
	if cfg.WindowMode == "tray" && oldMode != "tray" {
		runtime.WindowHide(a.ctx)
		a.windowShown = false
	} else if cfg.WindowMode == "floating" && oldMode != "floating" {
		runtime.WindowShow(a.ctx)
		a.windowShown = true
	}

	a.stopPolling()
	a.startPolling()
	return nil
}

func (a *App) SaveWindowWidth(width int) {
	a.mu.Lock()
	a.config.WindowWidth = width
	a.mu.Unlock()
	_ = a.writeConfig()
}

func (a *App) GetQuotaData() QuotaMonitorData {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.cached != nil {
		return *a.cached
	}
	return QuotaMonitorData{LastError: "API Key 未配置，请在设置中填入"}
}

// --- 网络请求 ---

func (a *App) fetchQuota() (*QuotaMonitorData, error) {
	a.mu.RLock()
	apiKey := a.config.APIKey
	a.mu.RUnlock()

	if apiKey == "" {
		return nil, fmt.Errorf("API Key 未配置")
	}

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", "https://open.bigmodel.cn/api/monitor/usage/quota/limit", nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Authorization", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("网络请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API 错误 %d: %s", resp.StatusCode, string(body))
	}

	var raw apiRawResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	if !raw.Success {
		return nil, fmt.Errorf("API 返回失败: %s", raw.Msg)
	}

	result := &QuotaMonitorData{Level: raw.Data.Level}
	tokenIdx := 0
	for _, limit := range raw.Data.Limits {
		switch limit.Type {
		case "TIME_LIMIT":
			result.McpCurrent = limit.Current
			result.McpTotal = limit.Usage
			result.McpLeft = limit.Remaining
			if limit.Usage > 0 {
				result.McpUsagePct = float64(limit.Current) / float64(limit.Usage) * 100
			}
		case "TOKENS_LIMIT":
			if tokenIdx == 0 {
				result.FiveHourUsedPct = limit.Percentage
				result.FiveHourLeftPct = 100 - limit.Percentage
				if limit.NextResetMs > 0 {
					t := time.Unix(limit.NextResetMs/1000, 0)
					result.NextRefreshTime = t.Format("15:04")
				}
			} else if tokenIdx == 1 {
				result.WeeklyUsedPct = limit.Percentage
				result.WeeklyLeftPct = 100 - limit.Percentage
			}
			tokenIdx++
		}
	}
	return result, nil
}

// --- 定时轮询 ---

func (a *App) startPolling() {
	a.mu.RLock()
	interval := time.Duration(a.config.RefreshInterval) * time.Minute
	apiKey := a.config.APIKey
	a.mu.RUnlock()

	if apiKey == "" {
		a.emitQuota(QuotaMonitorData{LastError: "API Key 未配置，请在设置中填入"})
		return
	}

	a.ticker = time.NewTicker(interval)
	go func() {
		a.doFetch()
		for {
			select {
			case <-a.ticker.C:
				a.doFetch()
			case <-a.stopCh:
				return
			}
		}
	}()
}

func (a *App) stopPolling() {
	if a.ticker != nil {
		a.ticker.Stop()
	}
	select {
	case a.stopCh <- struct{}{}:
	default:
	}
}

func (a *App) doFetch() {
	data, err := a.fetchQuota()
	if err != nil {
		errData := QuotaMonitorData{LastError: err.Error()}
		a.mu.Lock()
		a.cached = &errData
		a.mu.Unlock()
		a.emitQuota(errData)
		return
	}
	data.LastError = ""
	a.mu.Lock()
	a.cached = data
	a.mu.Unlock()
	a.emitQuota(*data)
}

func (a *App) emitQuota(data QuotaMonitorData) {
	runtime.EventsEmit(a.ctx, "update_quota", data)
	a.updateSystrayTooltip(data)
}

func (a *App) RefreshNow() {
	go a.doFetch()
}

func (a *App) SetWindowSize(width int, height int) {
	runtime.WindowSetSize(a.ctx, width, height)
}

// --- 开机自启动 ---

const autoStartRegKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
const autoStartValName = "GLMQuotaMonitor"

func exePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe
	}
	return resolved
}

func (a *App) SetAutoStart(enable bool) error {
	exe := exePath()
	if exe == "" {
		return fmt.Errorf("无法获取程序路径")
	}
	if enable {
		cmd := exec.Command("reg", "add", autoStartRegKey, "/v", autoStartValName, "/d", fmt.Sprintf(`"%s"`, exe), "/f")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("设置自启动失败: %s: %w", string(out), err)
		}
	} else {
		cmd := exec.Command("reg", "delete", autoStartRegKey, "/v", autoStartValName, "/f")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("取消自启动失败: %s: %w", string(out), err)
		}
	}
	return nil
}

func (a *App) IsAutoStart() bool {
	cmd := exec.Command("reg", "query", autoStartRegKey, "/v", autoStartValName)
	return cmd.Run() == nil
}

// --- 系统托盘 ---

// generateTrayIcon 生成 16x16 蓝色圆形 ICO
func generateTrayIcon() []byte {
	const w, h = 16, 16

	header := []byte{0, 0, 1, 0, 1, 0}

	pixels := make([]byte, w*h*4)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := ((h-1-y)*w + x) * 4
			dx := float64(x) - 7.5
			dy := float64(y) - 7.5
			if dx*dx+dy*dy <= 49 {
				pixels[i+0] = 0xFA
				pixels[i+1] = 0xB4
				pixels[i+2] = 0x89
				pixels[i+3] = 0xFF
			}
		}
	}

	bih := make([]byte, 40)
	binary.LittleEndian.PutUint32(bih[0:4], 40)
	binary.LittleEndian.PutUint32(bih[4:8], uint32(w))
	binary.LittleEndian.PutUint32(bih[8:12], uint32(h*2))
	binary.LittleEndian.PutUint16(bih[12:14], 1)
	binary.LittleEndian.PutUint16(bih[14:16], 32)

	andRowSize := ((w + 31) / 32) * 4
	andMask := make([]byte, andRowSize*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := float64(x) - 7.5
			dy := float64(y) - 7.5
			if dx*dx+dy*dy > 49 {
				andMask[y*andRowSize+x/8] |= 1 << (7 - uint(x%8))
			}
		}
	}

	dataSize := len(bih) + len(pixels) + len(andMask)

	dir := make([]byte, 16)
	dir[0] = byte(w)
	dir[1] = byte(h)
	binary.LittleEndian.PutUint16(dir[4:6], 1)
	binary.LittleEndian.PutUint16(dir[6:8], 32)
	binary.LittleEndian.PutUint32(dir[8:12], uint32(dataSize))
	binary.LittleEndian.PutUint32(dir[12:16], uint32(len(header)+len(dir)))

	result := make([]byte, 0, len(header)+len(dir)+dataSize)
	result = append(result, header...)
	result = append(result, dir...)
	result = append(result, bih...)
	result = append(result, pixels...)
	result = append(result, andMask...)
	return result
}

func (a *App) showWindow() {
	runtime.WindowShow(a.ctx)
	a.windowShown = true
	if a.mShow != nil {
		a.mShow.SetTitle("隐藏窗口")
		a.mShow.SetTooltip("隐藏悬浮窗")
	}
}

func (a *App) hideWindow() {
	runtime.WindowHide(a.ctx)
	a.windowShown = false
	if a.mShow != nil {
		a.mShow.SetTitle("显示窗口")
		a.mShow.SetTooltip("显示悬浮窗")
	}
}

func (a *App) toggleWindow() {
	if a.windowShown {
		a.hideWindow()
	} else {
		a.showWindow()
	}
}

func (a *App) startSystray() {
	a.mu.Lock()
	if a.systrayReady {
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()

	go systray.Run(func() {
		systray.SetIcon(generateTrayIcon())
		systray.SetTooltip("GLM 使用量监控")

		a.mShow = systray.AddMenuItem("隐藏窗口", "隐藏悬浮窗")
		systray.AddSeparator()
		a.mRefresh = systray.AddMenuItem("立即刷新", "刷新数据")
		a.mSettings = systray.AddMenuItem("设置", "打开设置")
		systray.AddSeparator()
		a.mExit = systray.AddMenuItem("退出", "退出程序")

		a.mu.Lock()
		a.systrayReady = true
		a.mu.Unlock()

		// 左键点击托盘图标
		systray.SetOnTapped(func() {
			go a.showWindow()
		})

		// 菜单事件
		go func() {
			for {
				select {
				case <-a.mShow.ClickedCh:
					a.toggleWindow()
				case <-a.mRefresh.ClickedCh:
					a.doFetch()
				case <-a.mSettings.ClickedCh:
					runtime.EventsEmit(a.ctx, "show_settings")
					a.showWindow()
				case <-a.mExit.ClickedCh:
					systray.Quit()
					runtime.Quit(a.ctx)
				}
			}
		}()
	}, func() {
		a.mu.Lock()
		a.systrayReady = false
		a.mu.Unlock()
	})
}

func (a *App) stopSystray() {
	a.mu.RLock()
	ready := a.systrayReady
	a.mu.RUnlock()
	if ready {
		systray.Quit()
	}
}

func (a *App) updateSystrayTooltip(data QuotaMonitorData) {
	a.mu.RLock()
	ready := a.systrayReady
	a.mu.RUnlock()
	if !ready {
		return
	}

	var tip string
	if data.LastError != "" {
		tip = "GLM 使用量监控: " + data.LastError
	} else {
		tip = fmt.Sprintf("GLM 使用量监控 | 本轮 %s%% | 刷新 %s",
			fmt.Sprintf("%.1f", data.FiveHourUsedPct),
			data.NextRefreshTime)
	}
	systray.SetTooltip(tip)
}
