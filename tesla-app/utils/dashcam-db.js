const DB_NAME = 'tesla_dashcam.db'
const DB_PATH = `_doc/${DB_NAME}`

let db = null

function isApp() {
  return typeof plus !== 'undefined' && plus.sqlite
}

function openDB() {
  return new Promise((resolve, reject) => {
    if (!isApp()) return reject(new Error('SQLite only available on App'))
    if (db) return resolve(db)
    plus.sqlite.openDatabase({
      name: DB_NAME,
      path: DB_PATH,
      success(e) {
        db = e
        resolve(db)
      },
      fail: reject
    })
  })
}

function executeSql(sql, params = []) {
  return new Promise((resolve, reject) => {
    if (!isApp()) return reject(new Error('SQLite only available on App'))
    plus.sqlite.executeSql({
      name: DB_NAME,
      sql,
      values: params,
      success: resolve,
      fail: reject
    })
  })
}

function selectSql(sql, params = []) {
  return new Promise((resolve, reject) => {
    if (!isApp()) return reject(new Error('SQLite only available on App'))
    plus.sqlite.selectSql({
      name: DB_NAME,
      sql,
      values: params,
      success: resolve,
      fail: reject
    })
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
  return executeSql(
    `INSERT INTO dashcam_events (vin, event_type, event_time, duration, latitude, longitude, thumbnail, imported, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    [vin, event_type, event_time, duration, latitude, longitude, thumbnail, imported || 0, created_at || Date.now()]
  ).then(res => {
    if (res && res.insertId) return res.insertId
    return selectSql('SELECT last_insert_rowid() as id').then(r => r[0].id)
  })
}

export function getEvents(options = {}) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve([])
  }
  const { eventType, limit = 50, offset = 0, startTime, endTime } = options
  let sql = 'SELECT * FROM dashcam_events WHERE 1=1'
  const params = []
  if (eventType) {
    sql += ' AND event_type = ?'
    params.push(eventType)
  }
  if (startTime) {
    sql += ' AND event_time >= ?'
    params.push(startTime)
  }
  if (endTime) {
    sql += ' AND event_time <= ?'
    params.push(endTime)
  }
  sql += ' ORDER BY event_time DESC LIMIT ? OFFSET ?'
  params.push(limit, offset)
  return selectSql(sql, params)
}

export function getEventById(id) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve(null)
  }
  return selectSql('SELECT * FROM dashcam_events WHERE id = ?', [id]).then(events => {
    if (!events || !events.length) return null
    const event = events[0]
    return selectSql('SELECT * FROM dashcam_videos WHERE event_id = ?', [id]).then(videos => {
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
  const keys = Object.keys(fields)
  const values = keys.map(k => fields[k])
  const setClause = keys.map(k => `${k} = ?`).join(', ')
  return executeSql(`UPDATE dashcam_events SET ${setClause} WHERE id = ?`, [...values, id])
}

export function deleteEvent(id) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve()
  }
  return executeSql('DELETE FROM dashcam_videos WHERE event_id = ?', [id]).then(() => {
    return executeSql('DELETE FROM dashcam_events WHERE id = ?', [id])
  })
}

export function insertVideo(video) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve(null)
  }
  const { event_id, camera, file_path, duration, file_size } = video
  return executeSql(
    `INSERT INTO dashcam_videos (event_id, camera, file_path, duration, file_size) VALUES (?, ?, ?, ?, ?)`,
    [event_id, camera, file_path, duration, file_size]
  ).then(res => {
    if (res && res.insertId) return res.insertId
    return selectSql('SELECT last_insert_rowid() as id').then(r => r[0].id)
  })
}

export function getVideosByEventId(eventId) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve([])
  }
  return selectSql('SELECT * FROM dashcam_videos WHERE event_id = ?', [eventId])
}

export function insertTrack(track) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve(null)
  }
  const { vin, latitude, longitude, speed, timestamp } = track
  return executeSql(
    `INSERT INTO vehicle_tracks (vin, latitude, longitude, speed, timestamp) VALUES (?, ?, ?, ?, ?)`,
    [vin, latitude, longitude, speed, timestamp]
  ).then(res => {
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
  const sql = `INSERT INTO vehicle_tracks (vin, latitude, longitude, speed, timestamp) VALUES (?, ?, ?, ?, ?)`
  const tasks = tracks.map(t => executeSql(sql, [t.vin, t.latitude, t.longitude, t.speed, t.timestamp]))
  return Promise.all(tasks)
}

export function getTracks(options = {}) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve([])
  }
  const { vin, startTime, endTime } = options
  let sql = 'SELECT * FROM vehicle_tracks WHERE 1=1'
  const params = []
  if (vin) {
    sql += ' AND vin = ?'
    params.push(vin)
  }
  if (startTime) {
    sql += ' AND timestamp >= ?'
    params.push(startTime)
  }
  if (endTime) {
    sql += ' AND timestamp <= ?'
    params.push(endTime)
  }
  sql += ' ORDER BY timestamp ASC'
  return selectSql(sql, params)
}

export function findNearestTrack(vin, timestamp, windowMs = 120000) {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve(null)
  }
  return selectSql(
    `SELECT * FROM vehicle_tracks WHERE vin = ? AND timestamp >= ? AND timestamp <= ? ORDER BY ABS(timestamp - ?) ASC LIMIT 1`,
    [vin, timestamp - windowMs, timestamp + windowMs, timestamp]
  ).then(rows => (rows && rows.length ? rows[0] : null))
}

export function getStorageStats() {
  if (!isApp()) {
    console.warn('SQLite only available on App')
    return Promise.resolve({ eventCount: 0, videoCount: 0, trackCount: 0, totalSize: 0 })
  }
  return selectSql('SELECT COUNT(*) as count FROM dashcam_events').then(([e]) => {
    return selectSql('SELECT COUNT(*) as count FROM dashcam_videos').then(([v]) => {
      return selectSql('SELECT COUNT(*) as count FROM vehicle_tracks').then(([t]) => {
        return selectSql('SELECT COALESCE(SUM(file_size), 0) as total FROM dashcam_videos').then(([s]) => {
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
    'SELECT id FROM dashcam_events WHERE event_type = ? AND event_time < ?',
    ['recent', cutoff]
  ).then(events => {
    if (!events || !events.length) return 0
    const ids = events.map(e => e.id)
    const placeholders = ids.map(() => '?').join(',')
    return executeSql(`DELETE FROM dashcam_videos WHERE event_id IN (${placeholders})`, ids).then(() => {
      return executeSql(`DELETE FROM dashcam_events WHERE id IN (${placeholders})`, ids).then(() => ids.length)
    })
  })
}
