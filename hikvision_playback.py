import cv2
import os
import sys
import time
from datetime import datetime, timedelta

# --- EXTRACTED CREDENTIALS & NVR SETTINGS ---
NVR_IP = "192.168.29.72"
PORT = 554
USERNAME = "cv"
PASSWORD_ENCODED = "Chakravue%402025"  # URL-encoded '@' as %40

# Target Date: 4 days ago
TARGET_DATE = (datetime.now() - timedelta(days=4)).strftime("%Y%m%d")

CHANNELS = {
    1: "101",
    2: "201",
    4: "401"
}

def get_rtsp_playback_url(track_id, date_str):
    """
    Hikvision Playback URL (without trailing slash):
    rtsp://user:pass@ip:port/Streaming/tracks/<track_id>?starttime=YYYYMMDDTHHMMSSZ&endtime=YYYYMMDDTHHMMSSZ
    """
    start_time = f"{date_str}T000000Z"
    end_time = f"{date_str}T235959Z"
    return f"rtsp://{USERNAME}:{PASSWORD_ENCODED}@{NVR_IP}:{PORT}/Streaming/tracks/{track_id}?starttime={start_time}&endtime={end_time}"

def get_rtsp_live_url(track_id):
    """Live stream RTSP URL"""
    return f"rtsp://{USERNAME}:{PASSWORD_ENCODED}@{NVR_IP}:{PORT}/Streaming/Channels/{track_id}"

def run_player(initial_channel=1, playback_mode=True):
    current_channel_num = initial_channel
    
    # Force TCP transport for maximum RTSP stability
    os.environ["OPENCV_FFMPEG_CAPTURE_OPTIONS"] = "rtsp_transport;tcp"

    print("=" * 65)
    print("       HIKVISION NVR CONTROLLER - 4-DAY PLAYBACK & LIVE")
    print(f"  Target Date : {TARGET_DATE} (4 Days Ago)")
    print(f"  NVR Address : {NVR_IP}:{PORT}")
    print(f"  User        : {USERNAME}")
    print("=" * 65)
    print("KEYBOARD CONTROLS:")
    print("  [SPACE]     : Pause / Resume (Peek mode)")
    print("  [1]         : 1x Normal Speed")
    print("  [2]         : 2x Fast Forward")
    print("  [4]         : 4x Fast Forward")
    print("  [C]         : Switch Camera Channel (1 -> 2 -> 4)")
    print("  [L]         : Toggle between 4-Day Playback and Live")
    print("  [S]         : Save Peek Snapshot Image to current folder")
    print("  [Q] or [ESC]: Quit")
    print("=" * 65)

    speed_multiplier = 1
    paused = False
    is_playback = playback_mode

    cv2.namedWindow("Hikvision NVR Player", cv2.WINDOW_NORMAL)
    cv2.resizeWindow("Hikvision NVR Player", 1280, 720)

    while True:
        track_id = CHANNELS[current_channel_num]
        if is_playback:
            url = get_rtsp_playback_url(track_id, TARGET_DATE)
            mode_label = f"PLAYBACK: {TARGET_DATE} (4 Days Ago)"
        else:
            url = get_rtsp_live_url(track_id)
            mode_label = "LIVE FEED"

        print(f"\n[Connecting] Channel {current_channel_num} (Track {track_id}) -> {mode_label}...")
        
        cap = cv2.VideoCapture(url, cv2.CAP_FFMPEG)
        cap.set(cv2.CAP_PROP_BUFFERSIZE, 2)

        # Try to read the initial frame
        ret, frame = (False, None)
        if cap.isOpened():
            ret, frame = cap.read()

        if not ret or frame is None:
            print(f"[Notice] No recorded stream found for Channel {current_channel_num} on {TARGET_DATE}.")
            print("Press [C] to switch to another channel, [L] for Live view, or [Q] to quit.")
            
            # Show a helpful placeholder screen in the window
            err_img = 20 * (cv2.imread(None) if False else 1)
            import numpy as np
            err_img = np.zeros((720, 1280, 3), dtype=np.uint8)
            cv2.putText(err_img, f"No recordings on {TARGET_DATE} for Channel {current_channel_num}", (180, 320),
                        cv2.FONT_HERSHEY_SIMPLEX, 1.0, (0, 0, 255), 2)
            cv2.putText(err_img, "Press 'C' to switch channel (Try 1 or 4), 'L' for Live, or 'Q' to quit", (160, 380),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.8, (255, 255, 255), 2)
            cv2.imshow("Hikvision NVR Player", err_img)
            
            wait_key = cv2.waitKey(0) & 0xFF
            cap.release()
            if wait_key in [ord('q'), ord('Q'), 27]:
                break
            elif wait_key in [ord('c'), ord('C')]:
                ch_list = list(CHANNELS.keys())
                curr_idx = ch_list.index(current_channel_num)
                current_channel_num = ch_list[(curr_idx + 1) % len(ch_list)]
                continue
            elif wait_key in [ord('l'), ord('L')]:
                is_playback = not is_playback
                continue
            else:
                continue

        print(f"[Streaming] Successfully playing Channel {current_channel_num} ({mode_label})")
        paused = False

        while True:
            if not paused:
                # Fast forward frame skipping
                if speed_multiplier == 2:
                    cap.grab()
                elif speed_multiplier == 4:
                    cap.grab()
                    cap.grab()
                    cap.grab()

                ret, frame = cap.read()
                if not ret or frame is None:
                    print("\n[INFO] End of stream reached.")
                    break

            # Render HUD overlay
            display = frame.copy()
            # Top-left info card
            cv2.rectangle(display, (15, 15), (500, 135), (0, 0, 0), -1)
            cv2.rectangle(display, (15, 15), (500, 135), (0, 220, 0), 2)

            status_text = "PAUSED (PEEK)" if paused else f"PLAYING ({speed_multiplier}x SPEED)"
            status_color = (0, 165, 255) if paused else (0, 255, 0)

            cv2.putText(display, f"CAMERA: Channel {current_channel_num} (Track {track_id})", (25, 45),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.7, (255, 255, 255), 2)
            cv2.putText(display, f"MODE  : {mode_label}", (25, 78),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.65, (0, 255, 255), 2)
            cv2.putText(display, f"STATE : {status_text}", (25, 115),
                        cv2.FONT_HERSHEY_SIMPLEX, 0.75, status_color, 2)

            cv2.imshow("Hikvision NVR Player", display)

            # Frame delay calculation for 1x, 2x, 4x
            if paused:
                delay = 80
            elif speed_multiplier == 1:
                delay = 25
            elif speed_multiplier == 2:
                delay = 12
            else:
                delay = 2

            key = cv2.waitKey(delay) & 0xFF

            if key in [ord('q'), ord('Q'), 27]:  # Quit
                cap.release()
                cv2.destroyAllWindows()
                print("\nExited.")
                sys.exit(0)

            elif key == ord(' '):  # Pause / Peek
                paused = not paused
                print(f"[Peek] {'PAUSED (Freeze Frame)' if paused else 'RESUMED'}")

            elif key == ord('1'):  # 1x
                speed_multiplier = 1
                paused = False
                print("[Speed] 1x Normal")

            elif key == ord('2'):  # 2x
                speed_multiplier = 2
                paused = False
                print("[Speed] 2x Fast Forward")

            elif key == ord('4'):  # 4x
                speed_multiplier = 4
                paused = False
                print("[Speed] 4x High Speed")

            elif key in [ord('s'), ord('S')]:  # Snapshot
                if frame is not None:
                    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
                    snap_name = f"peek_ch{current_channel_num}_{timestamp}.jpg"
                    cv2.imwrite(snap_name, frame)
                    print(f"[Saved] Snapshot saved -> {snap_name}")

            elif key in [ord('c'), ord('C')]:  # Switch Channel
                ch_list = list(CHANNELS.keys())
                curr_idx = ch_list.index(current_channel_num)
                current_channel_num = ch_list[(curr_idx + 1) % len(ch_list)]
                print(f"[Switch] Loading Channel {current_channel_num}...")
                break

            elif key in [ord('l'), ord('L')]:  # Toggle Live vs Playback
                is_playback = not is_playback
                print(f"[Toggle] Switched to {'Playback' if is_playback else 'Live'}")
                break

        cap.release()

if __name__ == "__main__":
    initial_ch = 1
    if len(sys.argv) > 1:
        try:
            initial_ch = int(sys.argv[1])
        except ValueError:
            pass
    run_player(initial_channel=initial_ch, playback_mode=True)
