# Proposal: Perbaikan Fitur "Continue Session" pada enowX

## 1. Ringkasan Masalah
Pada UI enowX, ketika sebuah session agen (khususnya **OMP**) telah berakhir (`stopped`/`done`/`exited`), muncul tombol **Continue** di toolbar sesi dan opsi **Continue** di menu konteks.
Namun ketika tombol tersebut diklik, muncul error pada status bar:
```text
session.continue: this session's command can't be continued: start it again in a new session
```

---

## 2. Root Cause Analysis (Penyebab Utama)

### A. Definisi Preset Agent (`core/agent/presets.go`)
Di `core/agent/presets.go`, entri untuk `omp` hanya terdaftar dasar tanpa konfigurasi sesi lanjutan:
```go
{
    ID: "omp", Name: "OMP", Command: "omp",
    Homepage: "https://omp.sh",
},
```
Padahal CLI `omp` memiliki opsi bawaan untuk melanjutkan sesi:
- `-c, --continue` : Continue previous session
- `-r, --resume=<value>` : Resume a session by ID prefix or path
- `--auto-approve` : Auto-approve all tool calls (skip approval prompts)

### B. Mekanisme Relaunch di Daemon (`daemon/create.go` & `daemon/persist.go`)
1. Ketika sesi dibuat (`daemon/create.go:286`), flag `relaunch` ditentukan oleh:
   ```go
   relaunch: len(preset.Continue) > 0,
   ```
   Karena preset `omp` tidak memiliki `Continue`, maka `meta.Relaunch` disimpan bernilai `false`.
2. Ketika user menekan tombol **Continue**, daemon menjalankan method `session.continue` (`daemon/persist.go:317`):
   ```go
   if !meta.Relaunch {
       return nil, errors.New("this session's command can't be continued: start it again in a new session")
   }
   ```
   Daemon menolak mengeksekusi relaunch karena `!meta.Relaunch`.

### C. Mismatch Logika di UI (`ui/session.go`)
Fungsi `canContinue` di `ui/session.go`:
```go
func canContinue(s proto.SessionInfo) bool {
    return s.Exited && s.Agent != "" && s.Agent != agent.Custom
}
```
UI mengasumsikan bahwa **semua** sesi agent yang exited dapat di-continue, tanpa memeriksa apakah preset agen tersebut sebenarnya memiliki kapabilitas continue (`len(preset.Continue) > 0`). Hal ini menimbulkan inkonsistensi: tombol Continue ditampilkan, tetapi pasti gagal saat diklik jika agen tidak mendukung continue.

---

## 3. Rencana Solusi (Action Plan)

### Rung 1: Dukung `omp` di Preset (`core/agent/presets.go` & `docs/agents.md`)
Menambahkan opsi continue, resume, dan skip permissions pada preset OMP:
```go
{
    ID: "omp", Name: "OMP", Command: "omp",
    SkipPermissions: []string{"--auto-approve"},
    Continue:        []string{"--continue"},
    Resume:          []string{"--resume"},
    Homepage:        "https://omp.sh",
},
```

### Rung 2: Perbaiki Validasi UI `canContinue` (`ui/session.go`)
Pastikan UI hanya menampilkan tombol/menu Continue untuk:
1. Shell (`s.Agent == agent.Shell`)
2. Agen yang terdaftar di preset dan memiliki `len(preset.Continue) > 0`

Dengan begitu:
- Sesi `omp` sekarang **bisa di-continue secara mulus**.
- Agen lain yang memang tidak memiliki fitur resume/continue tidak akan memunculkan tombol yang berujung error bagi user.

### Rung 3: Verifikasi & Test
1. Jalankan unit test `core/agent`: `go test ./core/agent`
2. Jalankan unit test `daemon`: `go test ./daemon`
3. Jalankan unit test `ui`: `go test ./ui`
4. Pastikan `go test ./proto` tetap valid.
