// ── Temperature sensors ────────────────────────────────────────────────────
export const TAG_TEMP_LIVING_ROOM   = '14'
export const TAG_TEMP_KIDS_OFFICE   = '15'
export const TAG_TEMP_HALLWAY       = '16'
export const TAG_TEMP_EDUARDO_ROOM  = '17'
export const TAG_TEMP_MAIN_ROOM     = '18'
export const TAG_TEMP_HOT_WATER     = '19'
export const TAG_TEMP_FILIPE_ROOM   = '22'
export const TAG_TEMP_OUTSIDE       = '25'
export const TAG_TEMP_MEN_CAVE      = '35'

// ── PC ─────────────────────────────────────────────────────────────────────
export const TAG_PC_CPU_LOAD        = '40'
export const TAG_PC_CPU_TEMP        = '41'
export const TAG_PC_GPU_LOAD        = '42'
export const TAG_PC_GPU_TEMP        = '43'

// ── Boiler ─────────────────────────────────────────────────────────────────
export const TAG_BOILER_HIGH_LIMIT  = '1'
export const TAG_BOILER_DELTA       = '2'
export const TAG_BOILER_WATER_TEMP  = '3'
export const TAG_BOILER_IMPULSION   = '4'
export const TAG_BOILER_STATE       = '6'
export const TAG_BOILER_EXHAUST     = '8'
export const TAG_BOILER_RETURN      = '9'

// ── Gate ───────────────────────────────────────────────────────────────────
export const TAG_GATE = 26

// ── Power (instantaneous W) ────────────────────────────────────────────────
export const TAG_POWER_TOTAL      = 10
export const TAG_POWER_TV         = 12
export const TAG_POWER_CAPTUR     = 20
export const TAG_POWER_SERVER     = 23
export const TAG_POWER_HOT_WATER  = 27
export const TAG_POWER_PC         = 29
export const TAG_POWER_DISHWASHER = 38
export const TAG_POWER_FRONTERA   = 44
export const TAG_POWER_WASHING    = 47
export const TAG_POWER_TOASTER    = 50

export const TAG_POWER_INSTANTANEOUS = [
    TAG_POWER_TOTAL,
    TAG_POWER_TV,
    TAG_POWER_CAPTUR,
    TAG_POWER_SERVER,
    TAG_POWER_HOT_WATER,
    TAG_POWER_PC,
    TAG_POWER_DISHWASHER,
    TAG_POWER_FRONTERA,
    TAG_POWER_WASHING,
    TAG_POWER_TOASTER
]

// ── Power device names ────────────────────────────────────────────────────
export const TAG_POWER_TOTAL_NAME      = 'total'
export const TAG_POWER_TV_NAME         = 'tv'
export const TAG_POWER_CAPTUR_NAME     = 'captur'
export const TAG_POWER_SERVER_NAME     = 'server'
export const TAG_POWER_HOT_WATER_NAME  = 'hotwater'
export const TAG_POWER_PC_NAME         = 'pc'
export const TAG_POWER_DISHWASHER_NAME = 'dishwasher'
export const TAG_POWER_FRONTERA_NAME   = 'frontera'
export const TAG_POWER_WASHING_NAME    = 'washing'
export const TAG_POWER_TOASTER_NAME    = 'toaster'

export const TAG_POWER_ID_NAMES: { [key: number]: string } = {
    [TAG_POWER_TOTAL]: TAG_POWER_TOTAL_NAME,
    [TAG_POWER_TV]: TAG_POWER_TV_NAME,
    [TAG_POWER_CAPTUR]: TAG_POWER_CAPTUR_NAME,
    [TAG_POWER_SERVER]: TAG_POWER_SERVER_NAME,
    [TAG_POWER_HOT_WATER]: TAG_POWER_HOT_WATER_NAME,
    [TAG_POWER_PC]: TAG_POWER_PC_NAME,
    [TAG_POWER_DISHWASHER]: TAG_POWER_DISHWASHER_NAME,
    [TAG_POWER_FRONTERA]: TAG_POWER_FRONTERA_NAME,
    [TAG_POWER_WASHING]: TAG_POWER_WASHING_NAME,
    [TAG_POWER_TOASTER]: TAG_POWER_TOASTER_NAME,
}


// ── Energy (cumulative Wh / Wm) ────────────────────────────────────────────
export const TAG_ENERGY_TOTAL       = 11
export const TAG_ENERGY_TV          = 13
export const TAG_ENERGY_CAPTUR      = 21
export const TAG_ENERGY_SERVER_ROOM = 24
export const TAG_ENERGY_PC          = 30
export const TAG_ENERGY_HOT_WATER   = 28
export const TAG_ENERGY_DISHWASHER  = 39
export const TAG_ENERGY_FRONTERA    = 45
export const TAG_ENERGY_WASHING     = 48

// ── Car ────────────────────────────────────────────────────────────────────
export const TAG_CAR_CAPTUR_BATTERY = 33
export const TAG_CAR_CAPTUR_PLUG    = 34
