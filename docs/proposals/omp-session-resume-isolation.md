# Proposal Perbaikan: Isolasi Percakapan OMP (Resume by Conversation ID)

## 1. Masalah yang Terjadi
Ketika pengguna memiliki beberapa sesi OMP di direktori yang sama (misalnya proyek non-git seperti `Research`), menekan tombol **Continue** pada sesi mana pun menyebabkan sesi tersebut memuat percakapan yang **sama persis** dengan sesi terakhir.

### Penyebab Akar:
1. `omp` (dan `claude`) secara bawaan menyimpan sesi percakapan per direktori kerja (`CWD`).
2. Perintah `omp --continue` hanya melanjutkan sesi percakapan dengan timestamp paling akhir di direktori tersebut.
3. enowX sebelumnya hanya mengeksekusi `omp --continue` generik saat me-relaunch sesi, tanpa menyertakan ID percakapan spesifik (`--resume <UUID>`).

---

## 2. Rencana Arsitektur & Perbaikan (Opsi 1: Seperti Claude Code)

### A. Komponen 1: Pembaca Transcript OMP (`core/transcript/omp.go`)
1. **Pencarian File (`OMPFiles`)**:
   - `omp` menyimpan riwayat sesi di:
     `~/.omp/agent/sessions/--<path-with-dashes>--/<timestamp>_<uuid>.jsonl`
   - Setiap file `.jsonl` memiliki baris session:
     `{"type":"session","id":"<uuid>","cwd":"..."}`
   - `OMPFiles(home, dir string) []Found` mencari dan mengembalikan daftar file percakapan OMP di direktori `dir`, diurutkan berdasarkan `ModTime` terbaru.
2. **Parser Percakapan (`ompParser`)**:
   - Membaca baris-baris JSONL OMP:
     - `session`: mengekstrak session UUID.
     - `model_change`: memperbarui model aktif (`r.Model`).
     - `message`: mengekstrak role `user` dan `assistant` (termasuk blok teks, thinking, dan tool calls).
3. **Pendaftaran di `core/transcript/agents.go`**:
   - Daftarkan `"omp": {ompFound, NewOMPReader}` sehingga `transcript.Supports("omp")` bernilai `true`.
   - Ini sekaligus mengaktifkan fitur **Chat View** untuk OMP di UI enowX!

### B. Komponen 2: Pelacakan ID Percakapan & Continue Sesi (`daemon/persist.go`)
1. Ketika sesi di-continue di `daemon/persist.go` (`continueSession`):
   - Daemon memeriksa apakah sesi memiliki transcript file yang sudah diklaim via `s.transcriptPath(id)`.
   - Jika ada transcript file OMP (atau Claude), daemon mengekstrak ID percakapan UUID dari file tersebut.
   - Perintah relaunch disesuaikan dari `omp --continue` menjadi:
     ```bash
     omp --resume <UUID> --auto-approve ...
     ```
2. Jika belum ada transcript file atau ID tidak ditemukan, fallback tetap ke `omp --continue`.

---

## 3. Dampak & Keuntungan
1. **Isolasi 100%**: Sesi 1 akan resume percakapan Sesi 1; Sesi 2 akan resume percakapan Sesi 2, meskipun berada di folder fisik yang sama persis tanpa Git worktree.
2. **Efisiensi & Standar enowX**:
   - Tidak ada codebloat; memanfaatkan pola yang sudah matang di `core/transcript` (arsitektur sama dengan `claude` dan `codex`).
   - Kode tetap ringkas dan mengikuti prinsip Go standar repository ini.
3. **Bonus Nilai Tambah**:
   - Sesi OMP kini mendukung **Chat View UI** di samping Terminal view.
