package com.purohit.contracts.i18n

import com.purohit.contracts.i18n.models.*

interface StringContract {
    val language: AppLanguage
    val auth: AuthStrings
    val catalog: CatalogStrings
    val samagri: SamagriStrings
    val dispatch: DispatchStrings
    val temple: TempleStrings
    val cancellation: CancellationStrings
    val common: CommonStrings
    val error: ErrorStrings
}
