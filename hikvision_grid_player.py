import cv2
import os
import sys
import time
import threading
import numpy as np
from datetime import datetime, timedelta

# --- EXTRACTED CREDENTIALS & NVR SETTINGS ---
NVR_IP = "192.168.29.72"
PORT = 554
USERNAME = "cv"
PASSWORD_ENCODED = "Chakravue%402025"

# Target Date: 4 days ago
TARGET_DATE = (datetime.now() - timedelta(days=4)).strftime("%Y%m%d")
FORMATTED_DATE = (datetime.now() - timedelta(days=4)).strftime("%Y-%m-%d")

# 8 Channels
TOTAL_CHANNELS = 8
CHANNELS = list(range(1, TOTAL_CHANNELS + 1))

# Grid dimensions
GRID_COLS = 4
GRID_ROWS = 2
TILE_WIDTH = 320
TILE_HEIGHT = 240
GRID_CANVAS_WIDTH = GRID_COLS * TILE_WIDTH      # 1280 px
GRID_CANVAS_HEIGHT = GRID_ROWS * TILE_HEIGHT    # 480 px

# Timeline Bar geometry in Expanded View (1280x720)
TIMELINE_X1 = 60
TIMELINE_X2 = 1220
TIMELINE_WIDTH = TIMELINE_X2 - TIMELINE_X1       # 1160 px
TIMELINE_Y1 = 620
TIMELINE_Y2 = 655

os.environ["OPENCV_FFMPEG_CAPTURE_OPTIONS"] = "rtsp_transport;tcp"


class CameraStream:
    """Threaded live RTSP streamer for low-latency grid."""
    def __init__(self, channel_num):
        self.channel = channel_num
        self.url = f"rtsp://{USERNAME}:{PASSWORD_ENCODED}@{NVR_IP}:{PORT}/Streaming/Channels/{channel_num}02"
        self.frame = None
        self.running = True
        self.lock = threading.Lock()
        self.thread = threading.Thread(target=self._stream_loop, daemon=True)
        self.thread.start()

    def _stream_loop(self):
        while self.running:
            cap = cv2.VideoCapture(self.url, cv2.CAP_FFMPEG)
            cap.set(cv2.CAP_PROP_BUFFERSIZE, 1)
            if not cap.isOpened():
                time.sleep(2)
                continue
            while self.running and cap.isOpened():
                ret, frame = cap.read()
                if ret and frame is not None:
                    with self.lock:
                        self.frame = frame
                else:
                    time.sleep(0.01)
            cap.release()

    def get_frame(self):
        with self.lock:
            return self.frame.copy() if self.frame is not None else None

    def stop(self):
        self.running = False


class NVRMultiController:
    def __init__(self):
        self.streams = {ch: CameraStream(ch) for ch in CHANNELS}
        self.mode = "GRID"  # "GRID" or "EXPANDED"
        self.expanded_channel = 1
        
        # 4-Day Playback State
        self.is_playback = True   # Default directly to 4-day playback!
        self.current_seconds = 0  # 00:00:00 to 86399 (seconds of the day)
        self.speed = 1            # 1x, 2x, 4x
        self.paused = False
        self.playback_cap = None
        self.last_frame = None
        self.is_dragging = False
        self.need_reconnect = True
        self.last_tick_time = time.time()

    def seconds_to_hms(self, sec):
        sec = max(0, min(86399, int(sec)))
        h = sec // 3600
        m = (sec % 3600) // 60
        s = sec % 60
        return f"{h:02d}:{m:02d}:{s:02d}"

    def get_playback_url(self, ch, start_sec):
        track_id = f"{ch}01"
        hms_str = self.seconds_to_hms(start_sec).replace(":", "")
        start_time = f"{TARGET_DATE}T{hms_str}Z"
        end_time = f"{TARGET_DATE}T235959Z"
        return f"rtsp://{USERNAME}:{PASSWORD_ENCODED}@{NVR_IP}:{PORT}/Streaming/tracks/{track_id}?starttime={start_time}&endtime={end_time}"

    def seek_to_seconds(self, sec):
        self.current_seconds = max(0, min(86399, sec))
        self.need_reconnect = True
        print(f"[SEEK] Jumping to {FORMATTED_DATE} {self.seconds_to_hms(self.current_seconds)}")

    def on_mouse(self, event, x, y, flags, param):
        if self.mode == "GRID":
            if event == cv2.EVENT_LBUTTONDOWN:
                col = x // TILE_WIDTH
                row = y // TILE_HEIGHT
                if col < GRID_COLS and row < GRID_ROWS:
                    ch_clicked = row * GRID_COLS + col + 1
                    if ch_clicked in CHANNELS:
                        self.expand_camera(ch_clicked)

        elif self.mode == "EXPANDED":
            # 1. Back button click (top left)
            if event == cv2.EVENT_LBUTTONDOWN:
                if 15 <= x <= 220 and 15 <= y <= 55:
                    self.return_to_grid()
                    return

            # 2. Timeline Bar click or drag
            if event == cv2.EVENT_LBUTTONDOWN:
                if TIMELINE_X1 <= x <= TIMELINE_X2 and (TIMELINE_Y1 - 10) <= y <= (TIMELINE_Y2 + 25):
                    self.is_dragging = True
                    ratio = (x - TIMELINE_X1) / TIMELINE_WIDTH
                    target_sec = int(ratio * 86400)
                    self.seek_to_seconds(target_sec)

            elif event == cv2.EVENT_MOUSEMOVE and self.is_dragging:
                ratio = max(0.0, min(1.0, (x - TIMELINE_X1) / TIMELINE_WIDTH))
                target_sec = int(ratio * 86400)
                self.seek_to_seconds(target_sec)

            elif event == cv2.EVENT_LBUTTONUP:
                self.is_dragging = False

    def expand_camera(self, ch):
        self.expanded_channel = ch
        self.mode = "EXPANDED"
        self.is_playback = True  # Always start in 4-day playback
        self.speed = 1
        self.paused = False
        self.current_seconds = 0
        self.need_reconnect = True
        self.last_frame = None
        if self.playback_cap is not None:
            self.playback_cap.release()
            self.playback_cap = None
        print(f"\n[EXPANDED] Camera {ch} -> 4-DAY RECORDING ({FORMATTED_DATE})")
        print("  * Click or Drag the timeline at bottom to skip to any hour/minute!")

    def return_to_grid(self):
        self.mode = "GRID"
        if self.playback_cap is not None:
            self.playback_cap.release()
            self.playback_cap = None
        print("\n[GRID] Returned to 8-Camera Multi-View Grid.")

    def render_grid(self):
        tiles = []
        for ch in CHANNELS:
            frame = self.streams[ch].get_frame()
            if frame is not None:
                tile = cv2.resize(frame, (TILE_WIDTH, TILE_HEIGHT))
            else:
                tile = np.zeros((TILE_HEIGHT, TILE_WIDTH, 3), dtype=np.uint8)
                cv2.putText(tile, f"CAM {ch}: Connecting...", (20, TILE_HEIGHT // 2),
                            cv2.FONT_HERSHEY_SIMPLEX, 0.55, (0, 165, 255), 1)

            cv2.rectangle(tile, (0, 0), (TILE_WIDTH, 26), (0, 0, 0), -1)
            cv2.putText(tile, f"CAM {ch} (LIVE PREVIEW)", (8, 18),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.5, (0, 255, 0), 1)
            cv2.rectangle(tile, (0, 0), (TILE_WIDTH - 1, TILE_HEIGHT - 1), (50, 50, 50), 1)
            tiles.append(tile)

        row1 = np.hstack(tiles[0:4])
        row2 = np.hstack(tiles[4:8])
        grid = np.vstack([row1, row2])

        # Bottom banner
        banner = np.zeros((65, GRID_CANVAS_WIDTH, 3), dtype=np.uint8)
        cv2.putText(banner, f">> CLICK ANY CAMERA TILE TO VIEW 4-DAY PLAYBACK ({FORMATTED_DATE}) <<", (80, 27),
                    cv2.FONT_HERSHEY_SIMPLEX, 0.7, (0, 255, 255), 2)
        cv2.putText(banner, "Press keys [1-8] to Select Camera | [Q] to Exit", (420, 53),
                    cv2.FONT_HERSHEY_SIMPLEX, 0.55, (180, 180, 180), 1)

        return np.vstack([grid, banner])

    def render_expanded(self):
        ch = self.expanded_channel
        hms_now = self.seconds_to_hms(self.current_seconds)

        if self.is_playback:
            # Reconnect if user sought or changed channel
            if self.need_reconnect:
                url = self.get_playback_url(ch, self.current_seconds)
                if self.playback_cap is not None:
                    self.playback_cap.release()
                self.playback_cap = cv2.VideoCapture(url, cv2.CAP_FFMPEG)
                self.playback_cap.set(cv2.CAP_PROP_BUFFERSIZE, 2)
                self.need_reconnect = False
                self.last_tick_time = time.time()

            if not self.paused and self.playback_cap is not None:
                # Fast forward frame skipping
                if self.speed == 2:
                    self.playback_cap.grab()
                elif self.speed == 4:
                    self.playback_cap.grab()
                    self.playback_cap.grab()
                    self.playback_cap.grab()

                ret, read_frame = self.playback_cap.read()
                if ret and read_frame is not None:
                    self.last_frame = read_frame
                    # Advance internal clock counter
                    now = time.time()
                    elapsed = now - self.last_tick_time
                    self.last_tick_time = now
                    self.current_seconds += elapsed * self.speed
                elif self.last_frame is None:
                    err_frame = np.zeros((720, 1280, 3), dtype=np.uint8)
                    cv2.putText(err_frame, f"No 4-Day Recordings on {FORMATTED_DATE} for Camera {ch}", (180, 340),
                                cv2.FONT_HERSHEY_SIMPLEX, 0.9, (0, 0, 255), 2)
                    cv2.putText(err_frame, "Try Camera 1 or 4, or press 'L' for Live view, 'G' for Grid", (220, 400),
                                cv2.FONT_HERSHEY_SIMPLEX, 0.7, (255, 255, 255), 2)
                    return err_frame

            frame = self.last_frame.copy() if self.last_frame is not None else np.zeros((720, 1280, 3), dtype=np.uint8)
            frame = cv2.resize(frame, (1280, 720))
            mode_label = f"4-DAY RECORDING ({FORMATTED_DATE})"

        else:
            # Live Feed
            live_frame = self.streams[ch].get_frame()
            if live_frame is not None:
                frame = cv2.resize(live_frame, (1280, 720))
            else:
                frame = np.zeros((720, 1280, 3), dtype=np.uint8)
            mode_label = "LIVE STREAM"

        # ========================================================
        # 1. TOP PROMINENT DATE & TIME BANNER (ALWAYS VISIBLE)
        # ========================================================
        cv2.rectangle(frame, (0, 0), (1280, 65), (10, 10, 10), -1)
        cv2.line(frame, (0, 65), (1280, 65), (0, 255, 255), 2)

        # Back to grid button
        cv2.rectangle(frame, (15, 12), (180, 52), (0, 60, 180), -1)
        cv2.rectangle(frame, (15, 12), (180, 52), (255, 255, 255), 2)
        cv2.putText(frame, "< GRID VIEW", (25, 38),
                    cv2.FONT_HERSHEY_SIMPLEX, 0.65, (255, 255, 255), 2)

        # High-visibility DATE & TIME at the TOP
        date_title = f"DATE: {FORMATTED_DATE} (4 DAYS AGO)" if self.is_playback else "DATE: TODAY (LIVE)"
        time_title = f"TIME: {hms_now}" if self.is_playback else f"TIME: {datetime.now().strftime('%H:%M:%S')}"
        status_text = "PAUSED (PEEK)" if self.paused else f"PLAYING {self.speed}x"
        status_col = (0, 165, 255) if self.paused else (0, 255, 0)

        cv2.putText(frame, f"CAM {ch}  |  {date_title}  |  {time_title}", (210, 40),
                    cv2.FONT_HERSHEY_SIMPLEX, 0.8, (0, 255, 255), 2)
        cv2.putText(frame, f"[{status_text}]", (1050, 40),
                    cv2.FONT_HERSHEY_SIMPLEX, 0.75, status_col, 2)

        # ========================================================
        # 2. INTERACTIVE DRAGGABLE TIMELINE SCRUBBER (AT BOTTOM)
        # ========================================================
        if self.is_playback:
            # Timeline background container
            cv2.rectangle(frame, (0, 595), (1280, 720), (15, 15, 15), -1)
            cv2.line(frame, (0, 595), (1280, 595), (60, 60, 60), 1)

            # Draw 24-Hour Timeline Track Bar
            cv2.rectangle(frame, (TIMELINE_X1, TIMELINE_Y1), (TIMELINE_X2, TIMELINE_Y2), (50, 50, 50), -1)
            cv2.rectangle(frame, (TIMELINE_X1, TIMELINE_Y1), (TIMELINE_X2, TIMELINE_Y2), (180, 180, 180), 2)

            # Hour tick marks (every 2 hours: 0, 2, 4, 6... 24)
            for hour in range(0, 25, 2):
                tx = int(TIMELINE_X1 + (hour / 24.0) * TIMELINE_WIDTH)
                cv2.line(frame, (tx, TIMELINE_Y1), (tx, TIMELINE_Y2), (90, 90, 90), 1)
                cv2.putText(frame, f"{hour:02d}:00", (tx - 18, TIMELINE_Y1 - 8),
                            cv2.FONT_HERSHEY_SIMPLEX, 0.45, (160, 160, 160), 1)

            # Played progress fill (green)
            progress_ratio = max(0.0, min(1.0, self.current_seconds / 86400.0))
            played_x = int(TIMELINE_X1 + progress_ratio * TIMELINE_WIDTH)
            cv2.rectangle(frame, (TIMELINE_X1, TIMELINE_Y1), (played_x, TIMELINE_Y2), (0, 180, 0), -1)

            # Draggable Scrubber Needle / Cursor (bright red/yellow)
            cv2.circle(frame, (played_x, (TIMELINE_Y1 + TIMELINE_Y2) // 2), 12, (0, 255, 255), -1)
            cv2.circle(frame, (played_x, (TIMELINE_Y1 + TIMELINE_Y2) // 2), 12, (0, 0, 255), 2)

            # Current time label above cursor
            cv2.putText(frame, hms_now, (max(TIMELINE_X1, played_x - 35), TIMELINE_Y1 - 8),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.6, (0, 255, 255), 2)

            # Controls help text at very bottom
            help_text = "[DRAG / CLICK TIMELINE] Jump Time | [<- / ->] -/+5m | [W] +30m | [SPACE] Peek | [1,2,4] Speed | [S] Snap | [G] Grid"
            cv2.putText(frame, help_text, (40, 690),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.55, (220, 220, 220), 1)

        return frame

    def run(self):
        cv2.namedWindow("Hikvision NVR 4-Day Playback Controller", cv2.WINDOW_NORMAL)
        cv2.setMouseCallback("Hikvision NVR 4-Day Playback Controller", self.on_mouse)

        print("\n=======================================================")
        print("  HIKVISION NVR 4-DAY PLAYBACK & TIMELINE CONTROLLER")
        print(f"  Target Date: {FORMATTED_DATE} (October 24, 2026)")
        print("=======================================================")
        print("TIMELINE & NAVIGATION CONTROLS:")
        print("  * MOUSE: Click or DRAG the timeline bar to jump to any hour/minute!")
        print("  * [->] or [D] : Skip forward +5 Minutes")
        print("  * [<-] or [A] : Skip backward -5 Minutes")
        print("  * [W]         : Skip forward +30 Minutes")
        print("  * [SPACE]     : Pause / Peek (Freeze Frame)")
        print("  * [1] / [2] / [4] : 1x Normal, 2x Fast, 4x Fast Forward")
        print("  * [S]         : Save Snapshot of current frame")
        print("  * [G] or [ESC]: Return to Grid View")
        print("  * [Q]         : Quit Program")
        print("=======================================================\n")

        while True:
            if self.mode == "GRID":
                cv2.resizeWindow("Hikvision NVR 4-Day Playback Controller", GRID_CANVAS_WIDTH, GRID_CANVAS_HEIGHT + 65)
                display = self.render_grid()
                delay = 25
            else:
                cv2.resizeWindow("Hikvision NVR 4-Day Playback Controller", 1280, 720)
                display = self.render_expanded()
                delay = 80 if self.paused else (25 if self.speed == 1 else (10 if self.speed == 2 else 2))

            cv2.imshow("Hikvision NVR 4-Day Playback Controller", display)
            key = cv2.waitKey(delay) & 0xFF

            if key in [ord('q'), ord('Q')]:
                break

            elif key in [27, ord('g'), ord('G')]:  # ESC or G: Grid
                if self.mode == "EXPANDED":
                    self.return_to_grid()
                else:
                    break

            # Fast Seek Arrow / Letter Keys
            elif key in [ord('d'), ord('D'), 83]:  # Right Arrow or D: +5 minutes (300 sec)
                if self.mode == "EXPANDED":
                    self.seek_to_seconds(self.current_seconds + 300)

            elif key in [ord('a'), ord('A'), 81]:  # Left Arrow or A: -5 minutes (300 sec)
                if self.mode == "EXPANDED":
                    self.seek_to_seconds(self.current_seconds - 300)

            elif key in [ord('w'), ord('W')]:  # W: +30 minutes (1800 sec)
                if self.mode == "EXPANDED":
                    self.seek_to_seconds(self.current_seconds + 1800)

            # Speed controls
            elif key == ord('1'):
                self.speed = 1
                self.paused = False
                print("[Speed] 1x Normal")

            elif key == ord('2'):
                self.speed = 2
                self.paused = False
                print("[Speed] 2x Fast Forward")

            elif key == ord('4'):
                self.speed = 4
                self.paused = False
                print("[Speed] 4x High Speed")

            elif key == ord(' '):  # Pause / Peek
                if self.mode == "EXPANDED":
                    self.paused = not self.paused
                    print(f"[Peek] {'PAUSED (Freeze Frame)' if self.paused else 'RESUMED'}")

            elif key in [ord('s'), ord('S')]:  # Snapshot
                if self.mode == "EXPANDED" and self.last_frame is not None:
                    hms_clean = self.seconds_to_hms(self.current_seconds).replace(":", "-")
                    filename = f"snapshot_cam{self.expanded_channel}_{FORMATTED_DATE}_{hms_clean}.jpg"
                    cv2.imwrite(filename, self.last_frame)
                    print(f"[Saved Snapshot] -> {filename}")

            elif ord('1') <= key <= ord(str(TOTAL_CHANNELS)):
                num = int(chr(key))
                if self.mode == "GRID":
                    self.expand_camera(num)

        # Cleanup
        for s in self.streams.values():
            s.stop()
        if self.playback_cap is not None:
            self.playback_cap.release()
        cv2.destroyAllWindows()


if __name__ == "__main__":
    controller = NVRMultiController()
    controller.run()
