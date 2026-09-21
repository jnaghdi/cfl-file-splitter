//go:build windows

// Native Windows desktop UI. Standard-library-only, no external runtime.
package main

import (
	"cflsplit/internal/core"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	user                            = syscall.NewLazyDLL("user32.dll")
	kernel                          = syscall.NewLazyDLL("kernel32.dll")
	gdi                             = syscall.NewLazyDLL("gdi32.dll")
	shell                           = syscall.NewLazyDLL("shell32.dll")
	comdlg                          = syscall.NewLazyDLL("comdlg32.dll")
	ole                             = syscall.NewLazyDLL("ole32.dll")
	common                          = syscall.NewLazyDLL("comctl32.dll")
	createWindow                    = user.NewProc("CreateWindowExW")
	defWindow                       = user.NewProc("DefWindowProcW")
	send                            = user.NewProc("SendMessageW")
	setTextProc                     = user.NewProc("SetWindowTextW")
	getTextProc                     = user.NewProc("GetWindowTextW")
	getTextLen                      = user.NewProc("GetWindowTextLengthW")
	show                            = user.NewProc("ShowWindow")
	enable                          = user.NewProc("EnableWindow")
	move                            = user.NewProc("MoveWindow")
	messageBox                      = user.NewProc("MessageBoxW")
	hwnd, instance, font, titleFont uintptr
	scale                           = 1.0
	controls                        = map[int]uintptr{}
	events                          = make(chan event, 256)
	busy                            bool
	stop                            context.CancelFunc
	closing                         bool
	lastFolder                      string
	latestInstructions              string
	startTime                       time.Time
	lastSplitOptions                core.Options
)

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type msg struct {
	Window         uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             point
	Private        uint32
}
type wndClass struct {
	Size, Style                        uint32
	Proc                               uintptr
	ClsExtra, WinExtra                 int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	SmallIcon                          uintptr
}
type minmax struct{ Reserved, MaxSize, MaxPosition, MinTrackSize, MaxTrackSize point }
type ofn struct {
	Size                         uint32
	Owner, Instance              uintptr
	Filter, CustomFilter         *uint16
	MaxCustomFilter, FilterIndex uint32
	File                         *uint16
	MaxFile                      uint32
	FileTitle                    *uint16
	MaxFileTitle                 uint32
	InitialDir, Title            *uint16
	Flags                        uint32
	FileOffset, FileExtension    uint16
	DefExt                       *uint16
	CustomData, Hook             uintptr
	Template                     *uint16
	Reserved                     uintptr
	ReservedWord, FlagsEx        uint32
}
type browseInfo struct {
	Owner, Root     uintptr
	Display, Title  *uint16
	Flags           uint32
	Callback, Param uintptr
	Image           int32
}
type event struct {
	Kind     string
	Progress core.Progress
	Result   core.Result
	Err      error
}

const (
	sourceEdit    = 101
	sourceButton  = 102
	outputEdit    = 103
	outputButton  = 104
	modeCombo     = 105
	sizeEdit      = 106
	splitButton   = 107
	verifyButton  = 108
	joinButton    = 109
	cancelButton  = 110
	openButton    = 111
	copyButton    = 112
	helpButton    = 113
	titleLabel    = 201
	versionLabel  = 202
	subtitleLabel = 203
	sourceGroup   = 204
	sourceInfo    = 205
	outputGroup   = 206
	outputNote    = 207
	optionsGroup  = 208
	modeLabel     = 209
	sizeLabel     = 210
	mbLabel       = 211
	modeNote      = 212
	estimateLabel = 213
	progressBar   = 214
	statusLabel   = 215
	logEdit       = 216
	footerLabel   = 217
	child         = 0x40000000
	visible       = 0x10000000
	tabstop       = 0x00010000
	border        = 0x00800000
)

func ptr(s string) *uint16 { p, _ := syscall.UTF16PtrFromString(s); return p }
func up(p *uint16) uintptr { return uintptr(unsafe.Pointer(p)) }
func px(i int) uintptr     { return uintptr(int(float64(i) * scale)) }
func text(id int, s string) {
	if h := controls[id]; h != 0 {
		setTextProc.Call(h, up(ptr(s)))
	}
}
func value(id int) string {
	h := controls[id]
	l, _, _ := getTextLen.Call(h)
	if l > 1000000 {
		return ""
	}
	b := make([]uint16, l+1)
	getTextProc.Call(h, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	return syscall.UTF16ToString(b)
}
func alert(title, s string, flags uintptr) uintptr {
	r, _, _ := messageBox.Call(hwnd, up(ptr(s)), up(ptr(title)), flags)
	return r
}
func setEnabled(id int, on bool) {
	v := uintptr(0)
	if on {
		v = 1
	}
	enable.Call(controls[id], v)
}
func add(id int, class, label string, style, ex uintptr) uintptr {
	h, _, _ := createWindow.Call(ex, up(ptr(class)), up(ptr(label)), child|visible|style, 0, 0, 10, 10, hwnd, uintptr(id), instance, 0)
	controls[id] = h
	send.Call(h, 0x30, font, 1)
	return h
}
func position(id, x, y, w, h int) { move.Call(controls[id], px(x), px(y), px(w), px(h), 1) }
func layout() {
	if hwnd == 0 || controls[sourceEdit] == 0 {
		return
	}
	var r rect
	user.NewProc("GetClientRect").Call(hwnd, uintptr(unsafe.Pointer(&r)))
	w := int(float64(r.Right) / scale)
	h := int(float64(r.Bottom) / scale)
	position(titleLabel, 24, 15, w-220, 36)
	position(versionLabel, w-226, 24, 202, 24)
	position(subtitleLabel, 24, 54, w-48, 22)
	position(sourceGroup, 16, 83, w-32, 86)
	position(sourceEdit, 32, 110, w-161, 28)
	position(sourceButton, w-115, 109, 83, 30)
	position(sourceInfo, 32, 143, w-64, 20)
	position(outputGroup, 16, 177, w-32, 82)
	position(outputEdit, 32, 203, w-161, 28)
	position(outputButton, w-115, 202, 83, 30)
	position(outputNote, 32, 236, w-64, 18)
	position(optionsGroup, 16, 267, w-32, 137)
	position(modeLabel, 32, 290, w-282, 20)
	position(sizeLabel, w-230, 290, 195, 20)
	position(modeCombo, 32, 312, w-282, 150)
	position(sizeEdit, w-230, 311, 120, 28)
	position(mbLabel, w-100, 316, 68, 22)
	position(modeNote, 32, 346, w-64, 34)
	position(estimateLabel, 32, 383, w-64, 18)
	position(splitButton, 24, 419, 155, 33)
	position(verifyButton, 191, 419, 154, 33)
	position(joinButton, 357, 419, 142, 33)
	position(cancelButton, 511, 419, 98, 33)
	position(openButton, 621, 419, w-645, 33)
	position(progressBar, 24, 466, w-48, 17)
	position(statusLabel, 24, 491, w-48, 22)
	position(logEdit, 24, 520, w-48, h-586)
	position(copyButton, 24, h-50, 184, 30)
	position(helpButton, 220, h-50, 94, 30)
	position(footerLabel, 330, h-46, w-354, 28)
}

func initControls() {
	add(titleLabel, "STATIC", "CFL File Splitter", 0, 0)
	send.Call(controls[titleLabel], 0x30, titleFont, 1)
	add(versionLabel, "STATIC", "v"+core.Version+"  |  Local processing", 2, 0)
	add(subtitleLabel, "STATIC", "Prepare smaller uploads. Identify every part. Verify the reconstructed file.", 0, 0)
	add(sourceGroup, "BUTTON", "  1. Source file  ", 7, 0)
	add(sourceEdit, "EDIT", "", tabstop|0x80, 0x200)
	add(sourceButton, "BUTTON", "Browse...", tabstop, 0)
	add(sourceInfo, "STATIC", "Choose a file, or drag one onto this window. The original contents are not changed.", 0, 0)
	add(outputGroup, "BUTTON", "  2. Output parent folder  ", 7, 0)
	add(outputEdit, "EDIT", "", tabstop|0x80, 0x200)
	add(outputButton, "BUTTON", "Browse...", tabstop, 0)
	add(outputNote, "STATIC", "A new, uniquely named subfolder will be created for each completed split set.", 0, 0)
	add(optionsGroup, "BUTTON", "  3. Split settings  ", 7, 0)
	add(modeLabel, "STATIC", "Output format", 0, 0)
	add(sizeLabel, "STATIC", "Maximum finished part size", 0, 0)
	add(modeCombo, "COMBOBOX", "", tabstop|0x0003|0x00200000, 0)
	for _, s := range []string{"Auto (recommended) - readable text or lossless encoded text", "Encoded text (.txt) - any file; rejoin before analysis", "Readable UTF-8 only (.txt) - strict text validation", "Binary parts (.cflpart) - local reassembly"} {
		send.Call(controls[modeCombo], 0x143, 0, up(ptr(s)))
	}
	send.Call(controls[modeCombo], 0x14e, 0, 0)
	add(sizeEdit, "EDIT", "20", tabstop|0x2000|0x80, 0x200)
	send.Call(controls[sizeEdit], 0xc5, 4, 0)
	add(mbLabel, "STATIC", "MB", 0, 0)
	add(modeNote, "STATIC", "", 0, 0)
	add(estimateLabel, "STATIC", "", 0, 0)
	add(splitButton, "BUTTON", "Split & verify", tabstop|1, 0)
	add(verifyButton, "BUTTON", "Verify a set...", tabstop, 0)
	add(joinButton, "BUTTON", "Rejoin a set...", tabstop, 0)
	add(cancelButton, "BUTTON", "Cancel", tabstop, 0)
	setEnabled(cancelButton, false)
	add(openButton, "BUTTON", "Open output folder", tabstop, 0)
	setEnabled(openButton, false)
	add(progressBar, "msctls_progress32", "", 0, 0)
	send.Call(controls[progressBar], 0x406, 0, 1000)
	add(statusLabel, "STATIC", "Ready. Start with a small, non-sensitive test file.", 0, 0)
	add(logEdit, "EDIT", "", 0x4|0x40|0x800|0x1000|0x00200000|tabstop, 0x200)
	add(copyButton, "BUTTON", "Copy Claude instructions", tabstop, 0)
	setEnabled(copyButton, false)
	add(helpButton, "BUTTON", "Help", tabstop, 0)
	add(footerLabel, "STATIC", "No automatic uploads or telemetry. Hashes are not digital signatures.", 0, 0)
	estimate()
	layout()
	shell.NewProc("DragAcceptFiles").Call(hwnd, 1)
	user.NewProc("SetTimer").Call(hwnd, 1, 80, 0)
	logLine("Local application. Source files are read only; split sets are verified from disk before completion.")
	logLine("For Claude: uploads must be accepted and accessible as original bytes. Splitting does not bypass token, context or file-count limits.")
}
func getMode() string {
	n, _, _ := send.Call(controls[modeCombo], 0x147, 0, 0)
	switch n {
	case 1:
		return "base64"
	case 2:
		return "utf8"
	case 3:
		return "binary"
	default:
		return "auto"
	}
}

// Before a complete scan, Auto estimates using Base64 capacity. The actual
// count may differ, including because readable mode prefers line boundaries.
func estimateEncoding(enc string) string {
	if enc == "auto" {
		return "base64"
	}
	return enc
}
func maxBytes() (int64, error) {
	n, e := strconv.ParseInt(strings.TrimSpace(value(sizeEdit)), 10, 64)
	if e != nil || n < 1 || n > 2000 {
		return 0, errors.New("Enter a whole number from 1 to 2000 MB. Start with 20 MB for Claude.")
	}
	return n * 1000000, nil
}
func estimate() {
	if controls[modeNote] == 0 {
		return
	}
	switch getMode() {
	case "auto":
		text(modeNote, "Checks the full file, not its extension. Uses readable UTF-8 when suitable, otherwise lossless Base64. Encoded pieces need code-based reassembly; no bytes are removed.")
	case "base64":
		text(modeNote, "Base64 is transport, not readable document text: Claude must decode and rejoin it using code. Payload size grows by about one third; acceptance is not guaranteed.")
	case "utf8":
		text(modeNote, "Preserves original UTF-8 bytes and prefers line endings. Very long lines may be split. This mode is NOT CSV-record, JSON-object or PDF-page aware.")
	case "binary":
		text(modeNote, "Smallest transport representation. Intended for local joining; .cflpart is not a documented ordinary Claude upload type.")
	}
	max, e := maxBytes()
	if e != nil {
		text(estimateLabel, "Enter a maximum part size from 1 to 2000 MB.")
		return
	}
	st, e := os.Stat(strings.TrimSpace(value(sourceEdit)))
	if e != nil {
		text(estimateLabel, "20 MB = 20,000,000 bytes including the part header. No upload-size guarantee is implied.")
		return
	}
	cap, e := core.PayloadCapacity(max, estimateEncoding(getMode()))
	if e != nil {
		return
	}
	count := (st.Size() + cap - 1) / cap
	if count < 1 {
		count = 1
	}
	prefix := "Estimated"
	if getMode() == "utf8" {
		prefix = "Minimum estimated"
	} else if getMode() == "auto" {
		prefix = "Estimated if encoded (actual may differ)"
	}
	warning := ""
	if count > 18 {
		warning = "  WARNING: more than 18 parts (+ 2 helper files)."
	}
	text(estimateLabel, fmt.Sprintf("%s parts: %d  |  Maximum per part: %.0f MB.%s", prefix, count, float64(max)/1000000, warning))
	text(sourceInfo, fmt.Sprintf("%s  |  %s bytes (%.2f MB)", st.Name(), commas(st.Size()), float64(st.Size())/1000000))
}
func commas(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
func chooseSource(path string) {
	fi, e := os.Stat(path)
	if e != nil || !fi.Mode().IsRegular() {
		alert("Choose a file", "Select a regular file, not a folder or device.", 0x10)
		return
	}
	text(sourceEdit, path)
	if strings.TrimSpace(value(outputEdit)) == "" {
		text(outputEdit, filepath.Dir(path))
	}
	// A .txt or .log extension does NOT establish its encoding. Every newly
	// chosen source starts in Auto; the worker validates its actual bytes.
	send.Call(controls[modeCombo], 0x14e, 0, 0)
	estimate()
}
func fileDialog(save bool, initial string) string {
	b := make([]uint16, 32768)
	if initial != "" {
		u, _ := syscall.UTF16FromString(initial)
		copy(b, u)
	}
	filter := []uint16{'A', 'l', 'l', ' ', 'f', 'i', 'l', 'e', 's', 0, '*', '.', '*', 0, 0}
	o := ofn{Owner: hwnd, Filter: &filter[0], File: &b[0], MaxFile: uint32(len(b)), Flags: 0x00080000 | 0x00000008 | 0x00000800}
	o.Size = uint32(unsafe.Sizeof(o))
	proc := comdlg.NewProc("GetOpenFileNameW")
	o.Title = ptr("Choose a file to split")
	if save {
		o.Title = ptr("Choose a NEW output filename (existing files will never be overwritten)")
		proc = comdlg.NewProc("GetSaveFileNameW")
	} else {
		o.Flags |= 0x00001000
	}
	ok, _, _ := proc.Call(uintptr(unsafe.Pointer(&o)))
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(b)
}
func folderDialog(title string) string {
	display := make([]uint16, 32768)
	b := browseInfo{Owner: hwnd, Display: &display[0], Title: ptr(title), Flags: 0x0001 | 0x0040}
	id, _, _ := shell.NewProc("SHBrowseForFolderW").Call(uintptr(unsafe.Pointer(&b)))
	if id == 0 {
		return ""
	}
	defer ole.NewProc("CoTaskMemFree").Call(id)
	path := make([]uint16, 32768)
	ok, _, _ := shell.NewProc("SHGetPathFromIDListW").Call(id, uintptr(unsafe.Pointer(&path[0])))
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(path)
}
func logLine(s string) {
	h := controls[logEdit]
	if h == 0 {
		return
	}
	length, _, _ := getTextLen.Call(h)
	if length > 58000 {
		old := value(logEdit)
		if len(old) > 30000 {
			text(logEdit, old[len(old)-30000:])
		}
	}
	send.Call(h, 0xb1, ^uintptr(0), ^uintptr(0))
	send.Call(h, 0xc2, 0, up(ptr(time.Now().Format("15:04:05")+"  "+s+"\r\n")))
	send.Call(h, 0xb7, 0, 0)
}
func setBusy(b bool) {
	busy = b
	for _, id := range []int{sourceEdit, sourceButton, outputEdit, outputButton, modeCombo, sizeEdit, splitButton, verifyButton, joinButton} {
		setEnabled(id, !b)
	}
	setEnabled(cancelButton, b)
	setEnabled(copyButton, !b && latestInstructions != "")
	setEnabled(openButton, !b && lastFolder != "")
}
func run(kind string, work func(context.Context, core.Notify) (core.Result, error)) {
	if busy {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	stop = cancel
	setBusy(true)
	startTime = time.Now()
	send.Call(controls[progressBar], 0x402, 0, 0)
	text(statusLabel, kind+"...")
	logLine(kind + " started.")
	go func() {
		last := time.Time{}
		notify := func(p core.Progress) {
			if p.Message == "" && time.Since(last) < 120*time.Millisecond {
				return
			}
			last = time.Now()
			ev := event{Kind: "progress", Progress: p}
			if p.Message != "" {
				events <- ev
			} else {
				select {
				case events <- ev:
				default:
				}
			}
		}
		r, e := work(ctx, notify)
		events <- event{Kind: kind, Result: r, Err: e}
	}()
}
func startSplit() {
	if busy {
		return
	}
	src := strings.TrimSpace(value(sourceEdit))
	out := strings.TrimSpace(value(outputEdit))
	if src == "" {
		path := fileDialog(false, "")
		if path == "" {
			return
		}
		chooseSource(path)
		src = path
		out = value(outputEdit)
	}
	if out == "" {
		out = folderDialog("Choose output parent folder")
		if out == "" {
			return
		}
		text(outputEdit, out)
	}
	max, e := maxBytes()
	if e != nil {
		alert("Invalid size", e.Error(), 0x10)
		return
	}
	enc := getMode()
	cap, _ := core.PayloadCapacity(max, estimateEncoding(enc))
	if st, e := os.Stat(src); e == nil && (st.Size()+cap-1)/cap > 18 {
		if alert("Upload-count warning", "This split is estimated to produce more than 18 parts plus two helper TXT files. That exceeds the app's original 20-file planning budget; current account/workflow limits may differ. Multiple messages must not be assumed to bypass per-chat limits.\n\nYou can still split for local use, or adjust the size if your Claude workflow permits it. Continue?", 0x34) != 6 {
			return
		}
	}
	if enc == "base64" || enc == "auto" {
		message := "Encoded TXT pieces contain Base64, not readable document sections. Auto uses readable UTF-8 only when the full source passes validation, otherwise it uses lossless Base64. Claude must accept the uploads, have access to their raw bytes and use code execution to decode/rejoin encoded pieces. Large encoded text may exceed processing limits.\n\nNo source bytes are changed or removed. Test a small non-sensitive file first. Continue?"
		if alert("Upload transport", message, 0x34) != 6 {
			return
		}
	}
	launchSplit(core.Options{Source: src, OutputParent: out, Encoding: enc, MaxPartBytes: max})
}
func launchSplit(o core.Options) {
	lastSplitOptions = o
	// Clear the previous set's links, so a failed/cancelled retry cannot offer
	// stale instructions or appear to open the new result when it opens an old one.
	latestInstructions = ""
	lastFolder = ""
	run("Split", func(ctx context.Context, n core.Notify) (core.Result, error) {
		return core.Split(ctx, o, n)
	})
}
func offerEncodedRetry(e error) {
	text(statusLabel, "This file needs encoded text. The original contents have not been changed.")
	logLine("Readable-only mode is incompatible: " + e.Error())
	if closing {
		return
	}
	o := lastSplitOptions
	countWarning := ""
	if st, se := os.Stat(o.Source); se == nil {
		if cap, ce := core.PayloadCapacity(o.MaxPartBytes, "base64"); ce == nil {
			count := (st.Size() + cap - 1) / cap
			if count > 18 {
				countWarning = fmt.Sprintf("\n\nEstimated encoded parts: %d, plus two helper files. Check your account's file-count limits before uploading.", count)
			}
		}
	}
	message := e.Error() + "\n\nThe original contents have not been changed. Retry in Encoded text to preserve ALL original bytes, including NULs?\n\nThese TXT pieces contain Base64. Claude must use code execution to decode/rejoin them before analysis. Upload acceptance is not guaranteed." + countWarning
	if alert("Switch to Encoded text and retry?", message, 0x34) != 6 {
		return
	}
	o.Encoding = "base64"
	send.Call(controls[modeCombo], 0x14e, 1, 0)
	estimate()
	launchSplit(o)
}
func startVerify() {
	if busy {
		return
	}
	folder := folderDialog("Select the folder containing ONE complete split set")
	if folder == "" {
		return
	}
	run("Verify", func(ctx context.Context, n core.Notify) (core.Result, error) { return core.Verify(ctx, folder, n) })
}
func startJoin() {
	if busy {
		return
	}
	folder := folderDialog("Select the folder containing ONE complete split set")
	if folder == "" {
		return
	}
	// Inventory and payload checks run in the worker. The save name is deliberately
	// not derived from untrusted headers. Choose the desired original extension.
	dest := fileDialog(true, filepath.Join(folder, "reconstructed_file"))
	if dest == "" {
		return
	}
	run("Rejoin", func(ctx context.Context, n core.Notify) (core.Result, error) { return core.Join(ctx, folder, dest, n) })
}
func processEvents() {
	for i := 0; i < 200; i++ {
		select {
		case ev := <-events:
			if ev.Kind == "progress" {
				p := ev.Progress
				v := int64(0)
				if p.Total > 0 {
					v = int64(float64(p.Done) / float64(p.Total) * 1000)
				}
				if v > 1000 {
					v = 1000
				}
				send.Call(controls[progressBar], 0x402, uintptr(v), 0)
				text(statusLabel, fmt.Sprintf("%s  |  %.1f / %.1f MB  |  elapsed %s", p.Phase, float64(p.Done)/1000000, float64(p.Total)/1000000, time.Since(startTime).Round(time.Second)))
				if p.Message != "" {
					logLine(p.Message)
				}
				continue
			}
			if stop != nil {
				stop()
				stop = nil
			}
			setBusy(false)
			if ev.Err != nil {
				if errors.Is(ev.Err, context.Canceled) {
					text(statusLabel, "Cancelled. No completed result was published.")
					logLine("Cancelled; incomplete output cleanup attempted.")
				} else if ev.Kind == "Split" && errors.Is(ev.Err, core.ErrNotReadableUTF8) {
					offerEncodedRetry(ev.Err)
				} else {
					text(statusLabel, "Stopped: "+ev.Err.Error())
					logLine("ERROR: " + ev.Err.Error())
					if !closing {
						alert("Operation stopped", ev.Err.Error(), 0x10)
					}
				}
			} else {
				r := ev.Result
				lastFolder = r.Folder
				if ev.Kind == "Split" {
					latestInstructions = filepath.Join(r.Folder, "CLAUDE_INSTRUCTIONS.txt")
				}
				setBusy(false)
				send.Call(controls[progressBar], 0x402, 1000, 0)
				text(statusLabel, fmt.Sprintf("%s complete: %d parts verified | %s original bytes", ev.Kind, r.Parts, commas(r.Size)))
				if ev.Kind == "Split" {
					logLine("Actual payload encoding: " + r.Encoding)
				}
				logLine("SHA-256 verified: " + r.SHA256)
				if r.Output != "" {
					logLine("Reconstructed output: " + r.Output)
				} else {
					logLine("Parts folder: " + r.Folder)
				}
				if ev.Kind == "Split" && r.Parts > 18 {
					logLine("WARNING: more than 18 parts. Do not assume this entire set fits one Claude chat.")
				}
				if ev.Kind == "Split" && r.Encoding == "binary" {
					logLine("Binary parts are for local joining. Use Auto or Encoded text for TXT transport; no Claude upload acceptance is guaranteed.")
				} else if ev.Kind == "Split" && r.Parts <= 18 {
					logLine("For Claude, attach the TXT parts and the two CLAUDE_*.txt helper files. Do not upload the application ZIP/EXE.")
				}
			}
			if closing {
				user.NewProc("DestroyWindow").Call(hwnd)
			}
		default:
			return
		}
	}
}
func copyInstructions() {
	b, e := os.ReadFile(latestInstructions)
	if e != nil {
		alert("Instructions unavailable", e.Error(), 0x10)
		return
	}
	ok, _, _ := user.NewProc("OpenClipboard").Call(hwnd)
	if ok == 0 {
		return
	}
	defer user.NewProc("CloseClipboard").Call()
	user.NewProc("EmptyClipboard").Call()
	chars, _ := syscall.UTF16FromString(string(b))
	mem, _, _ := kernel.NewProc("GlobalAlloc").Call(0x2, uintptr(len(chars)*2))
	if mem == 0 {
		return
	}
	p, _, _ := kernel.NewProc("GlobalLock").Call(mem)
	if p == 0 {
		kernel.NewProc("GlobalFree").Call(mem)
		return
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(chars)), chars)
	kernel.NewProc("GlobalUnlock").Call(mem)
	if r, _, _ := user.NewProc("SetClipboardData").Call(13, mem); r == 0 {
		kernel.NewProc("GlobalFree").Call(mem)
	} else {
		logLine("Claude instructions copied to clipboard.")
	}
}
func openFolder() {
	if lastFolder != "" {
		shell.NewProc("ShellExecuteW").Call(hwnd, up(ptr("open")), up(ptr(lastFolder)), 0, 0, 1)
	}
}
func help() {
	alert("Using CFL File Splitter", `SPLIT
Choose a source file and output parent folder. Start at 20 MB. Split & verify performs a source hash scan, writes parts, and independently verifies the saved parts. The source contents are never rewritten.

MODES
Auto (recommended): checks the complete source, not its extension. Suitable UTF-8 becomes readable text; NULs, invalid UTF-8 or incomplete characters trigger lossless Base64 instead. No bytes are stripped or converted. Auto is a transport choice, not a document-type detector.
Readable UTF-8 only: strict mode; incompatible input offers an encoded-text retry. Text/log payloads remain readable. Prefers line endings; does not preserve complete CSV records, JSON objects or PDF pages.
Encoded text: any file becomes Base64 TXT pieces. They must be decoded and rejoined before processing; they are not ordinary readable documents.
Binary: efficient .cflpart transport for local use, not a documented ordinary Claude upload type.

CLAUDE
Upload TXT pieces plus CLAUDE_JOINER.txt and CLAUDE_INSTRUCTIONS.txt. Enable code execution. Try a small non-sensitive file. Raw attachment bytes are required. Upload acceptance, token limits and sandbox resources are NOT guaranteed; splitting does not bypass them. Binary evidence formats may still need specialist tools.

VERIFY / REJOIN
Select one parts folder. Internal metadata establishes ordering, even after renaming pieces. Missing, duplicate, mixed or corrupted parts stop the operation. Rejoin requires a NEW output path; existing files are never overwritten. Choose the original file extension yourself.

INTEGRITY
Checksums are not digital signatures, encryption, backups or proof of authorship. Keep the original hash independently. Only file content is preserved, not ACLs, alternate streams, original timestamps or chain of custody.

This utility has no automatic uploads, updater or telemetry. Windows interface: unsigned build; test locally before production use.`, 0x40)
}
func wndProc(h uintptr, m uint32, w, l uintptr) uintptr {
	switch m {
	case 1:
		hwnd = h
		initControls()
		return 0
	case 5:
		layout()
		return 0
	case 0x24:
		info := (*minmax)(unsafe.Pointer(l))
		info.MinTrackSize = point{int32(860 * scale), int32(680 * scale)}
		return 0
	case 0x113:
		processEvents()
		return 0
	case 0x111:
		id := int(w & 0xffff)
		code := (w >> 16) & 0xffff
		if id == modeCombo && code == 1 {
			estimate()
			return 0
		}
		if (id == sizeEdit || id == sourceEdit) && code == 0x300 {
			estimate()
			return 0
		}
		if code != 0 {
			return 0
		}
		switch id {
		case sourceButton:
			p := fileDialog(false, "")
			if p != "" {
				chooseSource(p)
			}
		case outputButton:
			p := folderDialog("Choose output parent folder")
			if p != "" {
				text(outputEdit, p)
			}
		case splitButton:
			startSplit()
		case verifyButton:
			startVerify()
		case joinButton:
			startJoin()
		case cancelButton:
			if stop != nil {
				stop()
				setEnabled(cancelButton, false)
				text(statusLabel, "Cancellation requested; cleaning up incomplete output...")
			}
		case openButton:
			openFolder()
		case copyButton:
			copyInstructions()
		case helpButton:
			help()
		}
		return 0
	case 0x233:
		defer shell.NewProc("DragFinish").Call(w)
		if busy {
			return 0
		}
		b := make([]uint16, 32768)
		n, _, _ := shell.NewProc("DragQueryFileW").Call(w, 0, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
		if n > 0 {
			chooseSource(syscall.UTF16ToString(b))
		}
		return 0
	case 0x10:
		if busy {
			if alert("Operation in progress", "Cancel the current operation and close when cleanup finishes?", 0x34) == 6 {
				closing = true
				stop()
			}
			return 0
		}
		user.NewProc("DestroyWindow").Call(h)
		return 0
	case 2:
		user.NewProc("KillTimer").Call(h, 1)
		user.NewProc("PostQuitMessage").Call(0)
		return 0
	}
	r, _, _ := defWindow.Call(h, uintptr(m), w, l)
	return r
}
func main() {
	runtime.LockOSThread()
	user.NewProc("SetProcessDPIAware").Call()
	dpi := user.NewProc("GetDpiForSystem")
	if dpi.Find() == nil {
		d, _, _ := dpi.Call()
		if d >= 96 && d <= 384 {
			scale = float64(d) / 96
		}
	}
	ole.NewProc("CoInitializeEx").Call(0, 2)
	defer ole.NewProc("CoUninitialize").Call()
	init := struct{ Size, Classes uint32 }{8, 0x20}
	common.NewProc("InitCommonControlsEx").Call(uintptr(unsafe.Pointer(&init)))
	instance, _, _ = kernel.NewProc("GetModuleHandleW").Call(0)
	makeFont := func(height, weight int) uintptr {
		h, _, _ := gdi.NewProc("CreateFontW").Call(uintptr(int64(-int(float64(height)*scale))), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 5, 0, up(ptr("Segoe UI")))
		return h
	}
	font = makeFont(15, 400)
	titleFont = makeFont(29, 600)
	icon, _, _ := user.NewProc("LoadIconW").Call(0, 32512)
	cursor, _, _ := user.NewProc("LoadCursorW").Call(0, 32512)
	class := wndClass{Proc: syscall.NewCallback(wndProc), Instance: instance, Icon: icon, SmallIcon: icon, Cursor: cursor, Background: 16, ClassName: ptr("CFLFileSplitterV1")}
	class.Size = uint32(unsafe.Sizeof(class))
	if ok, _, _ := user.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&class))); ok == 0 {
		alert("Startup error", "Could not register the application window.", 0x10)
		return
	}
	screenH, _, _ := user.NewProc("GetSystemMetrics").Call(1)
	desiredHeight := 790
	available := int(float64(screenH)/scale) - 45
	if available < desiredHeight {
		desiredHeight = available
	}
	if desiredHeight < 680 {
		desiredHeight = 680
	}
	h, _, _ := createWindow.Call(0, up(class.ClassName), up(ptr("CFL File Splitter - Split, Verify & Rejoin")), 0x00cf0000, 0x80000000, 0x80000000, px(980), px(desiredHeight), 0, 0, instance, 0)
	if h == 0 {
		alert("Startup error", "Could not create the application window.", 0x10)
		return
	}
	hwnd = h
	show.Call(hwnd, 1)
	user.NewProc("UpdateWindow").Call(hwnd)
	if len(os.Args) > 1 {
		chooseSource(os.Args[1])
	}
	var m msg
	for {
		r, _, _ := user.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if r == 0 || int32(r) == -1 {
			break
		}
		isDialog, _, _ := user.NewProc("IsDialogMessageW").Call(hwnd, uintptr(unsafe.Pointer(&m)))
		if isDialog == 0 {
			user.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&m)))
			user.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&m)))
		}
	}
	gdi.NewProc("DeleteObject").Call(font)
	gdi.NewProc("DeleteObject").Call(titleFont)
}
