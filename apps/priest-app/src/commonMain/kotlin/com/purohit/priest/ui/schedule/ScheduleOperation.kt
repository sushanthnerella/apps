package com.purohit.priest.ui.schedule

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Job
import kotlinx.coroutines.launch

/** Keep the controls busy until publishing and the subsequent refresh have both finished. */
internal fun CoroutineScope.publishThenRefresh(
    publish: suspend () -> Result<Unit>,
    refresh: suspend () -> Unit,
    setLoading: (Boolean) -> Unit,
    onFailure: (Throwable) -> Unit
): Job {
    setLoading(true)
    return launch {
        try {
            val result = publish()
            result.fold(onSuccess = { refresh() }, onFailure = onFailure)
        } finally {
            setLoading(false)
        }
    }
}
