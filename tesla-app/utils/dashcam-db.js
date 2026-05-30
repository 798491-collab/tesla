const DB_NAME = 'tesla_dashcam.db'
const DB_PATH = `_doc/${DB_NAME}`

let db = null
let dbOpened = false
let openPromise = null

function isApp() {
  return typeof plus !== 'undefined' && plus.sqlite
}

function esc(val) {
  if (val == null) return 'NULL'
  if (typeof val === 'number') return String(val)
  return "'" + String(val).replace(/'/g, "''") + "'"
}

function openDB() {
  if (dbOpened && db) {
    return Promise.resolve(db)
  }

  if (openPromise) {
    return openPromise
  }

  openPromise = new Promise((resolve, reject) => {
    if (!isApp()) {
      openPromise = null
      return reject(new Error('SQLite only available on App'))
    }

    plus.sqlite.openDatabase({
      name: DB_NAME,
      path: DB_PATH,
      success(e) {
        db = e
        dbOpened = true
        openPromise = null
        console.log('[DashcamDB] database opened successfully')
        resolve(db)
      },
      fail(err) {
        console.warn('[DashcamDB] openDatabase result:', JSON.stringify(err))
        if (err && err.code === -1402) {
          db = { name: DB_NAME }
          dbOpened = true
          openPromise = null
          console.log('[DashcamDB] -1402 = already open, reuse connection')
          resolve(db)
          return
        }
        openPromise = null
        reject(err)
      }
    })
  })

  return openPromise
}

function executeSql(sql) {
  return new Promise((resolve, reject) => {
    if (!isApp()) return reject(new Error('SQLite only available on App'))
    if (!dbOpened) {
      openDB().then(() => _doExecuteSql(sql, resolve, reject)).catch(reject)
      return
    }
    _doExecuteSql(sql, resolve, reject)
  })
}

function _doExecuteSql(sql, resolve, reject) {
  plus.sqlite.executeSql({
    name: DB_NAME,
    sql,
    success: resolve,
    fail(err) {
      console.error('[DashcamDB] executeSql failed:', JSON.stringify(err), 'sql:', sql.substring(0, 200))
      if (err && err.code === -1402) {
        dbOpened = true
        db = { name: DB_NAME }
        _doExecuteSql(sql, resolve, reject)
        return
      }
      reject(err)
    }
  })
}

export function selectSql(sql) {
  return new Promise((resolve, reject) => {
    if (!isApp()) return reject(new Error('SQLite only available on App'))
    if (!dbOpened) {
      openDB().then(() => _doSelectSql(sql, resolve, reject)).catch(reject)
      return
    }
    _doSelectSql(sql, resolve, reject)
  })
}

function _doSelectSql(sql, resolve, reject) {
  plus.sqlite.selectSql({
    name: DB_NAME,
    sql,
    success: resolve,
    fail(err) {
      console.error('[DashcamDB] selectSql failed:', JSON.stringify(err), 'sql:', sql.substring(0, 200))
      if (err && err.code === -1402) {
        dbOpened = true
        db = { name: DB_NAME }
        _doSelectSql(sql, resolve, reject)
        return
      }
      reject(err)
    }
  })
}

export function initDB() {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve()
  }
  return openDB().then(() => {
    return executeSql(`
      CREATE TABLE IF NOT EXISTS dashcam_events (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        vin TEXT,
        event_type TEXT,
        event_time INTEGER,
        duration INTEGER,
        latitude REAL,
        longitude REAL,
        thumbnail TEXT,
        imported INTEGER DEFAULT 0,
        created_at INTEGER
      )
    `)
  }).then(() => {
    return executeSql(`
      CREATE TABLE IF NOT EXISTS dashcam_videos (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        event_id INTEGER,
        camera TEXT,
        file_path TEXT,
        duration INTEGER,
        file_size INTEGER,
        FOREIGN KEY (event_id) REFERENCES dashcam_events(id)
      )
    `)
  }).then(() => {
    return executeSql(`
      CREATE TABLE IF NOT EXISTS vehicle_tracks (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        vin TEXT,
        latitude REAL,
        longitude REAL,
        speed REAL,
        timestamp INTEGER
      )
    `)
  })
}

export function closeDB() {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve()
  }
  return new Promise((resolve, reject) => {
    plus.sqlite.closeDatabase({
      name: DB_NAME,
      success() {
        db = null
        dbOpened = false
        openPromise = null
        resolve()
      },
      fail: reject
    })
  })
}

export function insertEvent(event) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve(null)
  }
  const { vin, event_type, event_time, duration, latitude, longitude, thumbnail, imported, created_at } = event
  const sql = `INSERT INTO dashcam_events (vin, event_type, event_time, duration, latitude, longitude, thumbnail, imported, created_at) VALUES (${esc(vin)}, ${esc(event_type)}, ${esc(event_time)}, ${esc(duration)}, ${esc(latitude)}, ${esc(longitude)}, ${esc(thumbnail)}, ${esc(imported)}, ${esc(created_at || Date.now())})`
  console.log('[DashcamDB] insertEvent sql:', sql.substring(0, 200))
  return executeSql(sql).then(res => {
    if (res && res.insertId) return res.insertId
    return selectSql('SELECT last_insert_rowid() as id').then(r => r[0].id)
  })
}

export function checkEventExists(eventTime, eventType) {
  if (!isApp()) {
    return Promise.resolve(false)
  }
  return selectSql(
    `SELECT id FROM dashcam_events WHERE event_time = ${esc(eventTime)} AND event_type = ${esc(eventType)} AND imported = 1 LIMIT 1`
  ).then(res => {
    return res && res.length > 0
  }).catch(() => false)
}

export function getEvents(options = {}) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve([])
  }
  const { eventType, limit = 50, offset = 0, startTime, endTime } = options
  let sql = 'SELECT * FROM dashcam_events WHERE imported = 1'
  if (eventType) {
    sql += ` AND event_type = ${esc(eventType)}`
  }
  if (startTime) {
    sql += ` AND event_time >= ${Number(startTime)}`
  }
  if (endTime) {
    sql += ` AND event_time <= ${Number(endTime)}`
  }
  const safeLimit = Math.max(1, parseInt(limit, 10) || 50)
  const safeOffset = Math.max(0, parseInt(offset, 10) || 0)
  sql += ` ORDER BY event_time DESC LIMIT ${safeLimit} OFFSET ${safeOffset}`
  console.log('[DashcamDB] getEvents sql:', sql)
  return selectSql(sql)
}

export function getEventsWithVideoCount(options = {}) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve([])
  }
  const { eventType, limit = 50, offset = 0, startTime, endTime } = options
  let sql = `SELECT e.*, COUNT(v.id) as video_count FROM dashcam_events e LEFT JOIN dashcam_videos v ON v.event_id = e.id WHERE e.imported = 1`
  if (eventType) {
    sql += ` AND e.event_type = ${esc(eventType)}`
  }
  if (startTime) {
    sql += ` AND e.event_time >= ${Number(startTime)}`
  }
  if (endTime) {
    sql += ` AND e.event_time <= ${Number(endTime)}`
  }
  sql += ` GROUP BY e.id`
  const safeLimit = Math.max(1, parseInt(limit, 10) || 50)
  const safeOffset = Math.max(0, parseInt(offset, 10) || 0)
  sql += ` ORDER BY e.event_time DESC LIMIT ${safeLimit} OFFSET ${safeOffset}`
  console.log('[DashcamDB] getEventsWithVideoCount sql:', sql)
  return selectSql(sql)
}

export function getEventById(id) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve(null)
  }
  const safeId = parseInt(id, 10)
  return selectSql(`SELECT * FROM dashcam_events WHERE id = ${safeId}`).then(events => {
    if (!events || !events.length) return null
    const event = events[0]
    return selectSql(`SELECT * FROM dashcam_videos WHERE event_id = ${safeId}`).then(videos => {
      event.videos = videos || []
      return event
    })
  })
}

export function updateEvent(id, fields) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve()
  }
  const safeId = parseInt(id, 10)
  const setClause = Object.keys(fields).map(k => `${k} = ${esc(fields[k])}`).join(', ')
  const sql = `UPDATE dashcam_events SET ${setClause} WHERE id = ${safeId}`
  console.log('[DashcamDB] updateEvent sql:', sql.substring(0, 200))
  return executeSql(sql)
}

export function deleteEvent(id) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve([])
  }
  const safeId = parseInt(id, 10)
  return selectSql(`SELECT file_path FROM dashcam_videos WHERE event_id = ${safeId}`).then(videos => {
    const paths = (videos || []).map(v => v.file_path).filter(Boolean)
    return executeSql(`DELETE FROM dashcam_videos WHERE event_id = ${safeId}`).then(() => {
      return executeSql(`DELETE FROM dashcam_events WHERE id = ${safeId}`).then(() => {
        console.log('[DashcamDB] deleteEvent id:', safeId, 'deleted', paths.length, 'video records')
        return paths
      })
    })
  })
}

export function insertVideo(video) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve(null)
  }
  const { event_id, camera, file_path, duration, file_size } = video
  const sql = `INSERT INTO dashcam_videos (event_id, camera, file_path, duration, file_size) VALUES (${esc(event_id)}, ${esc(camera)}, ${esc(file_path)}, ${esc(duration)}, ${esc(file_size)})`
  console.log('[DashcamDB] insertVideo sql:', sql.substring(0, 200))
  return executeSql(sql).then(res => {
    if (res && res.insertId) return res.insertId
    return selectSql('SELECT last_insert_rowid() as id').then(r => r[0].id)
  })
}

export function getVideosByEventId(eventId) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve([])
  }
  return selectSql(`SELECT * FROM dashcam_videos WHERE event_id = ${parseInt(eventId, 10)}`)
}

export function insertTrack(track) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve(null)
  }
  const { vin, latitude, longitude, speed, timestamp } = track
  const sql = `INSERT INTO vehicle_tracks (vin, latitude, longitude, speed, timestamp) VALUES (${esc(vin)}, ${esc(latitude)}, ${esc(longitude)}, ${esc(speed)}, ${esc(timestamp)})`
  return executeSql(sql).then(res => {
    if (res && res.insertId) return res.insertId
    return selectSql('SELECT last_insert_rowid() as id').then(r => r[0].id)
  })
}

export function insertTrackBatch(tracks) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve()
  }
  if (!tracks || !tracks.length) return Promise.resolve()
  const tasks = tracks.map(t => {
    const sql = `INSERT INTO vehicle_tracks (vin, latitude, longitude, speed, timestamp) VALUES (${esc(t.vin)}, ${esc(t.latitude)}, ${esc(t.longitude)}, ${esc(t.speed)}, ${esc(t.timestamp)})`
    return executeSql(sql)
  })
  return Promise.all(tasks)
}

export function getTracks(options = {}) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve([])
  }
  const { vin, startTime, endTime } = options
  let sql = 'SELECT * FROM vehicle_tracks WHERE 1=1'
  if (vin) {
    sql += ` AND vin = ${esc(vin)}`
  }
  if (startTime) {
    sql += ` AND timestamp >= ${Number(startTime)}`
  }
  if (endTime) {
    sql += ` AND timestamp <= ${Number(endTime)}`
  }
  sql += ' ORDER BY timestamp ASC'
  return selectSql(sql)
}

export function findNearestTrack(vin, timestamp, windowMs = 120000) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve(null)
  }
  const ts = Number(timestamp)
  return selectSql(
    `SELECT * FROM vehicle_tracks WHERE vin = ${esc(vin)} AND timestamp >= ${ts - windowMs} AND timestamp <= ${ts + windowMs} ORDER BY ABS(timestamp - ${ts}) ASC LIMIT 1`
  ).then(rows => (rows && rows.length ? rows[0] : null))
}

export function getStorageStats() {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve({ eventCount: 0, videoCount: 0, trackCount: 0, totalSize: 0 })
  }
  return selectSql('SELECT COUNT(*) as count FROM dashcam_events WHERE imported = 1').then(([e]) => {
    return selectSql('SELECT COUNT(*) as count FROM dashcam_videos v INNER JOIN dashcam_events e ON v.event_id = e.id WHERE e.imported = 1').then(([v]) => {
      return selectSql('SELECT COUNT(*) as count FROM vehicle_tracks').then(([t]) => {
        return selectSql('SELECT COALESCE(SUM(v.file_size), 0) as total FROM dashcam_videos v INNER JOIN dashcam_events e ON v.event_id = e.id WHERE e.imported = 1').then(([s]) => {
          return {
            eventCount: e.count,
            videoCount: v.count,
            trackCount: t.count,
            totalSize: s.total
          }
        })
      })
    })
  })
}

export function cleanOldRecentClips(days = 7) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve(0)
  }
  const cutoff = Date.now() - days * 24 * 60 * 60 * 1000
  return selectSql(
    `SELECT id FROM dashcam_events WHERE event_type = 'recent' AND event_time < ${cutoff}`
  ).then(events => {
    if (!events || !events.length) return 0
    const ids = events.map(e => e.id)
    const idList = ids.join(',')
    return executeSql(`DELETE FROM dashcam_videos WHERE event_id IN (${idList})`).then(() => {
      return executeSql(`DELETE FROM dashcam_events WHERE id IN (${idList})`).then(() => ids.length)
    })
  })
}

export function cleanPendingImports() {
  if (!isApp()) {
    return Promise.resolve(0)
  }
  return selectSql(
    'SELECT id FROM dashcam_events WHERE imported = 0'
  ).then(events => {
    if (!events || !events.length) return 0
    const ids = events.map(e => e.id)
    const idList = ids.join(',')
    return executeSql(`DELETE FROM dashcam_videos WHERE event_id IN (${idList})`).then(() => {
      return executeSql(`DELETE FROM dashcam_events WHERE id IN (${idList})`).then(() => {
        console.log('[DashcamDB] cleaned', ids.length, 'pending imports')
        return ids.length
      })
    })
  }).catch(() => 0)
}
