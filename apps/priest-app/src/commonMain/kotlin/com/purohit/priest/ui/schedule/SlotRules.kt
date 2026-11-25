package com.purohit.priest.ui.schedule

import com.purohit.priest.network.PriestSlotPayload
import kotlinx.datetime.LocalDate
import kotlinx.datetime.LocalDateTime

enum class SlotProblem { PAST_DATE, PAST_TIME, END_NOT_AFTER_START, OVERLAP }

/** Minutes since midnight from "HH:mm" (seconds, if present, are ignored). */
fun minutesOf(hhmm: String): Int = hhmm.take(5).split(":").let { it[0].toInt() * 60 + it[1].toInt() }

/** "HH:mm" for the wire format. */
fun hhmm(minutes: Int): String = "${(minutes / 60).toString().padStart(2, '0')}:${(minutes % 60).toString().padStart(2, '0')}"

/** 12-hour display, e.g. 540 -> "9:00 AM", 0 -> "12:00 AM", 780 -> "1:00 PM". */
fun format12h(minutes: Int): String {
    val h = minutes / 60
    val m = (minutes % 60).toString().padStart(2, '0')
    val h12 = if (h % 12 == 0) 12 else h % 12
    return "$h12:$m ${if (h < 12) "AM" else "PM"}"
}

/** Client-side check before publishing; the server re-checks past and overlap. Times are IST. */
fun validateSlot(date: LocalDate, startMin: Int, endMin: Int, nowIst: LocalDateTime, existing: List<PriestSlotPayload>): SlotProblem? {
    if (date < nowIst.date) return SlotProblem.PAST_DATE
    if (endMin <= startMin) return SlotProblem.END_NOT_AFTER_START
    if (date == nowIst.date && startMin <= nowIst.hour * 60 + nowIst.minute) return SlotProblem.PAST_TIME
    if (existing.any { startMin < minutesOf(it.end_time) && minutesOf(it.start_time) < endMin }) return SlotProblem.OVERLAP
    return null
}

fun PriestSlotPayload.isRemovable() = !is_booked && !is_held
