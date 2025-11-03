# Music Playlist Sync Tool

M3U8プレイリストとその音楽ファイルをUSBメモリに効率的に同期するGoプログラムです。

## 概要

このツールは以下の機能を提供します：

- **プレイリスト解析**: M3U8形式のプレイリストファイルを解析し、参照されている音楽ファイルを抽出
- **重複排除**: 複数のプレイリストで共通の楽曲がある場合、重複を排除して効率的に同期
- **パス変換**: 絶対パスから相対パスへの自動変換でUSBメモリでの再生に対応
- **rsync同期**: 高速で信頼性の高いrsyncを使用したファイル同期
- **自動クリーンアップ**: 不要になった古いプレイリストファイルの自動削除
- **ドライラン**: 実際の変更前に動作確認が可能

## 必要な環境

- **Go 1.16以上**
- **同期ツール**: rsync または rclone
- **macOS/Linux**: シンボリンクをサポートするOS

### 同期ツールのインストール

#### rsync（デフォルト）
```bash
# macOS (Homebrew)
brew install rsync

# Ubuntu/Debian
sudo apt-get install rsync

# CentOS/RHEL
sudo yum install rsync
```

#### rclone（オプション）
```bash
# macOS (Homebrew)
brew install rclone

# Ubuntu/Debian
sudo apt-get install rclone

# その他のOS
curl https://rclone.org/install.sh | sudo bash
```

## インストール

```bash
# リポジトリをクローン
git clone <repository-url>
cd sync_music

# ビルド
go build -o music-sync .
```

## 使用方法

### 基本的な使用例

```bash
# ドライラン（実際の変更は行わない）
./music-sync -playlist "~/Music/Playlists/*.m3u8" -usbRoot "/Volumes/UNTITLED" -dryrun

# 実際の同期実行（rsync使用）
./music-sync -playlist "~/Music/Playlists/*.m3u8" -usbRoot "/Volumes/UNTITLED"

# rcloneを使用した同期
./music-sync -playlist "~/Music/Playlists/*.m3u8" -usbRoot "/Volumes/UNTITLED" -sync rclone
```

### コマンドラインオプション

| オプション | 必須 | デフォルト | 説明 | 例 |
|-----------|------|-----------|------|-----|
| `-playlist` | ✓ | - | プレイリストファイルのGlobパターン | `"~/Playlists/*.m3u8"` |
| `-usbRoot` | ✓ | - | USBメモリのルートディレクトリ | `"/Volumes/UNTITLED"` |
| `-sync` | - | `rsync` | 同期ツール（rsync または rclone） | `rclone` |
| `-dryrun` | - | `false` | ドライランモード（変更を実行しない） | - |

### 使用例

```bash
# 特定のプレイリストのみ同期（rsync）
./music-sync -playlist "~/Music/Playlists/favorites.m3u8" -usbRoot "/Volumes/MyUSB"

# 複数のプレイリストを一括同期（rsync）
./music-sync -playlist "~/Music/Playlists/*.m3u8" -usbRoot "/Volumes/MyUSB"

# rcloneを使用した同期
./music-sync -playlist "~/Music/Playlists/*.m3u8" -usbRoot "/Volumes/MyUSB" -sync rclone

# ドライランで事前確認
./music-sync -playlist "~/Music/Playlists/*.m3u8" -usbRoot "/Volumes/MyUSB" -dryrun

# rclone + ドライラン
./music-sync -playlist "~/Music/Playlists/*.m3u8" -usbRoot "/Volumes/MyUSB" -sync rclone -dryrun
```

## 動作の詳細

### 1. プレイリスト解析
- M3U8ファイルを読み込み、`#EXTINF`タグと音楽ファイルパスを抽出
- Windows（`\r\n`）、Mac（`\r`）、Unix（`\n`）の改行形式に対応
- 絶対パスの音楽ファイルのみを対象として処理
- 制御文字の除去とパス正規化を実行

### 2. 重複排除
- 複数のプレイリストで参照される同一ファイルを特定
- 一意のファイルリストを作成して効率的な同期を実現

### 3. 一時ディレクトリ作成
- システムの一時ディレクトリにシンボリンクツリーを作成
- 元のディレクトリ構造を保持しながら同期用の構造を構築

### 4. ファイル同期
#### rsync使用時
- `-avL`オプション: アーカイブモード、詳細出力、シンボリンクの実体をコピー
- `--progress`: 進行状況表示
- `--delete`: 不要ファイルの削除

#### rclone使用時
- `sync`コマンド: 一方向同期
- `--copy-links`: シンボリンクの実体をコピー
- `--progress`: 進行状況表示
- `--delete-during`: 不要ファイルの削除

両方ともUSB内の`Music`ディレクトリに同期

### 5. プレイリスト変換
- 絶対パスを相対パス（`./`から始まる）に変換
- USB内でのプレイリスト再生に対応
- 元のメタデータ（`#EXTINF`タグ）を保持

### 6. クリーンアップ
- 現在のプレイリストセットに含まれない古いプレイリストファイルを削除
- USB容量の効率的な利用

## ディレクトリ構造

### 同期前（ローカル）
```
~/Music/
├── Playlists/
│   ├── favorites.m3u8
│   └── workout.m3u8
├── Artists/
│   ├── Artist1/
│   │   └── song1.mp3
│   └── Artist2/
│       └── song2.mp3
```

### 同期後（USB）
```
/Volumes/UNTITLED/
└── Music/
    ├── favorites.m3u8      # 相対パスに変換済み
    ├── workout.m3u8        # 相対パスに変換済み
    ├── Artists/
    │   ├── Artist1/
    │   │   └── song1.mp3
    │   └── Artist2/
    │       └── song2.mp3
```

## プレイリスト形式の変換例

### 変換前（ローカル）
```m3u8
#EXTM3U
#EXTINF:180,Artist1 - Song Title
/Users/username/Music/Artists/Artist1/song1.mp3
```

### 変換後（USB）
```m3u8
#EXTM3U
#EXTINF:180,Artist1 - Song Title
./Artists/Artist1/song1.mp3
```

## トラブルシューティング

### よくある問題

#### 1. プレイリストが見つからない
```
Error: no playlists matched: ~/Playlists/*.m3u8
```
**解決方法**: プレイリストファイルのパスとGlobパターンを確認してください。

#### 2. USBディレクトリにアクセスできない
```
Error: usbRoot required (e.g. --usbRoot /Volumes/UNTITLED)
```
**解決方法**: USBメモリがマウントされていることを確認し、正しいパスを指定してください。

#### 3. rsyncエラー
```
rsync error: some files/attrs were not transferred
```
**解決方法**: 
- ファイルの読み取り権限を確認
- USB容量不足の可能性を確認
- ドライランモードで事前確認

#### 4. 音楽ファイルが見つからない
```
src stat error /path/to/file.mp3: no such file or directory
```
**解決方法**: プレイリスト内のファイルパスが正しいか確認してください。

### デバッグ方法

1. **ドライランモードの使用**
   ```bash
   ./music-sync -playlist "*.m3u8" -usbRoot "/Volumes/USB" -dryrun
   ```

2. **ログ出力の確認**
   - 警告メッセージは処理を継続しますが、内容を確認してください
   - エラーメッセージは処理を停止する重要な問題を示します

3. **段階的なテスト**
   - 小さなプレイリストから開始
   - 徐々にファイル数を増やして問題を特定

## パフォーマンス

### 最適化のポイント

- **重複排除**: 同一ファイルの重複コピーを防止
- **rsync効率**: 変更されたファイルのみを転送
- **シンボリンク**: 一時的なファイルコピーを回避
- **並列処理**: rsyncの内部並列処理を活用
- **メモリ効率**: `strings.Builder`による効率的なファイル書き込み
- **処理最適化**: 不要な関数呼び出しと重複処理を削減

### 推奨事項

- 初回同期は時間がかかる場合があります
- 2回目以降は差分のみの高速同期
- USB 3.0以上の使用を推奨
- SSDタイプのUSBメモリで最適なパフォーマンス

## ライセンス

このプロジェクトはMITライセンスの下で公開されています。


