const CAMERA_MAP = {
  front: 'front',
  back: 'back',
  left_repeater: 'left_repeater',
  right_repeater: 'right_repeater'
}

const CLIP_TYPES = {
  RecentClips: 'recent',
  SavedClips: 'saved',
  SentryClips: 'sentry'
}

const REQUEST_CODE_SELECT_DIR = 1001

let _safResolve = null
let _safListenerReady = false
let _eventMap = {}
let _copyQueue = Promise.resolve()
let _copyQueueActive = false

function isApp() {
  return typeof plus !== 'undefined' && plus.android
}

function log(tag, ...args) {
  console.log(`[Dashcam:${tag}]`, ...args)
}

// 安全解析 Tesla 时间格式：2026-05-29_12-00-00
function parseEventTime(str) {
  const match = str.match(/^(\d{4})-(\d{2})-(\d{2})_(\d{2})-(\d{2})-(\d{2})$/)
  if (!match) return null
  return new Date(
    Number(match[1]),
    Number(match[2]) - 1,
    Number(match[3]),
    Number(match[4]),
    Number(match[5]),
    Number(match[6])
  )
}

// 从目录名解析事件时间（Saved/Sentry 的事件目录格式：2026-05-29_02-17-11）
function parseEventIdTime(name) {
  const match = name.match(/(\d{4}-\d{2}-\d{2})_(\d{2}-\d{2}-\d{2})/)
  if (!match) return null
  return parseEventTime(match[1] + '_' + match[2])
}

// 检查时间是否在筛选范围内
function isInRange(eventTime, minEventTime) {
  if (!minEventTime || minEventTime <= 0) return true
  return eventTime >= minEventTime
}

// 支持更多 Tesla 摄像头命名变体
function parseCamera(fileName) {
  const lower = fileName.toLowerCase()

  if (lower.includes('left_repeater') || lower.includes('leftpillar') || lower.includes('left_pillar')) {
    return 'left_repeater'
  }

  if (lower.includes('right_repeater') || lower.includes('rightpillar') || lower.includes('right_pillar')) {
    return 'right_repeater'
  }

  if (lower.includes('front')) {
    return 'front'
  }

  if (lower.includes('back') || lower.includes('rear')) {
    return 'back'
  }

  return null
}

function getMainActivity() {
  return plus.android.runtimeMainActivity()
}

function getLocalBasePath() {
  try {
    const main = getMainActivity()

    let dir = plus.android.invoke(
      main,
      'getExternalFilesDir',
      null
    )

    if (!dir) {
      dir = plus.android.invoke(
        main,
        'getFilesDir'
      )
    }

    if (!dir) {
      throw new Error('dir null')
    }

    const path = plus.android.invoke(
      dir,
      'getAbsolutePath'
    )

    return String(path)

  } catch (e) {
    log('PATH', 'getLocalBasePath failed', e.message || e)

    const main = getMainActivity()

    return (
      '/storage/emulated/0/Android/data/' +
      plus.android.invoke(main, 'getPackageName') +
      '/files'
    )
  }
}

function takePersistablePermission(main, uri) {
  try {
    const FLAG_GRANT_READ_URI_PERMISSION = 1
    const FLAG_GRANT_WRITE_URI_PERMISSION = 2
    const takeFlags = FLAG_GRANT_READ_URI_PERMISSION | FLAG_GRANT_WRITE_URI_PERMISSION

    const resolver = plus.android.invoke(main, 'getContentResolver')

    try {
      plus.android.invoke(resolver, 'takePersistableUriPermission', uri, takeFlags)
      log('PERM', 'takePersistableUriPermission OK')
    } catch (e) {
      log('PERM', 'takePersistableUriPermission FAILED (SAFE IGNORE)', e.message || '')
    }
  } catch (e) {
    log('PERM', 'takePersistablePermission FAILED (SAFE IGNORE)', e.message || '')
  }
}

function ensureSAFListener() {
  if (_safListenerReady) return
  _safListenerReady = true

  if (typeof plus !== 'undefined' && plus.globalEvent) {
    try {
      plus.globalEvent.addEventListener('activityResult', function(e) {
        if (e.requestCode === REQUEST_CODE_SELECT_DIR && _safResolve) {
          const resolve = _safResolve
          _safResolve = null
          if (e.resultCode === -1) {
            try {
              let uri = null
              if (e.data && typeof e.data.getData === 'function') {
                uri = e.data.getData()
              } else if (e.data) {
                uri = plus.android.invoke(e.data, 'getData')
              }
              const uriStr = uri ? String(uri.toString()) : null
              if (uri) {
                const main = getMainActivity()
                takePersistablePermission(main, uri)
              }
              resolve(uriStr)
            } catch (err) {
              log('GLOBAL', 'URI extraction failed', err.message || err)
              resolve(null)
            }
          } else {
            resolve(null)
          }
        }
      })
      log('GLOBAL', 'listener registered')
    } catch (e) {
      log('GLOBAL', 'register failed', e.message || e)
    }
  }
}

function overrideOnActivityResult(main, doResolve) {
  try {
    const origResult = main.onActivityResult
    main.onActivityResult = function(requestCode, resultCode, data) {
      if (requestCode === REQUEST_CODE_SELECT_DIR) {
        log('ACTIVITY', 'onActivityResult fired', 'resultCode=' + resultCode)
        if (resultCode === -1 && data) {
          try {
            const uri = data.getData()
            const uriStr = uri ? String(uri.toString()) : null
            log('ACTIVITY', 'got URI', uriStr)
            if (uri) {
              takePersistablePermission(main, uri)
            }
            doResolve(uriStr)
          } catch (err) {
            log('ACTIVITY', 'getData failed', err.message || err)
            doResolve(null)
          }
        } else {
          doResolve(null)
        }
      }
      if (typeof origResult === 'function') {
        try { origResult.call(main, requestCode, resultCode, data) } catch (e) {}
      }
    }
    return true
  } catch (e) {
    log('ACTIVITY', 'override failed', e.message || e)
    return false
  }
}

export function selectTeslaCamDir() {
  if (!isApp()) {
    console.warn('SAF only available on Android App')
    return Promise.resolve(null)
  }

  return new Promise((resolve) => {
    let resolved = false
    const doResolve = (value) => {
      if (resolved) return
      resolved = true
      _safResolve = null
      resolve(value)
    }

    try {
      const main = getMainActivity()
      const Intent = plus.android.importClass('android.content.Intent')
      const intent = new Intent('android.intent.action.OPEN_DOCUMENT_TREE')

      setTimeout(() => {
        log('TIMEOUT', 'SAF selection timeout after 120s')
        doResolve(null)
      }, 120000)

      overrideOnActivityResult(main, doResolve)
      ensureSAFListener()
      main.startActivityForResult(intent, REQUEST_CODE_SELECT_DIR)
      log('SAF', 'startActivityForResult sent')
    } catch (e) {
      log('SAF', 'startActivityForResult failed', e.message || e)
      doResolve(null)
    }
  })
}

function ensureUri(uri) {
  if (!uri) return null
  if (typeof uri === 'string') {
    const Uri = plus.android.importClass('android.net.Uri')
    return Uri.parse(uri)
  }
  return uri
}

function safeListFiles(dir, label) {
  try {
    log(label, 'listFiles() calling...')
    const t0 = Date.now()
    const files = dir.listFiles()
    const elapsed = Date.now() - t0
    if (!files) {
      log(label, 'listFiles() returned null', elapsed + 'ms')
      return null
    }
    log(label, 'listFiles() count=' + files.length, elapsed + 'ms')
    return files
  } catch (e) {
    log(label, 'listFiles() ERROR', e.message || e)
    return null
  }
}

export function scanTeslaCam(treeUri, minEventTime = 0) {
  if (!isApp()) {
    return Promise.resolve({ events: [] })
  }

  return new Promise((resolve) => {
    try {
      _eventMap = {}
      const _minEventTime = minEventTime

      const main = getMainActivity()
      const uri = ensureUri(treeUri)
      if (!uri) {
        log('SCAN', 'uri is null')
        resolve({ events: [] })
        return
      }

      log('SCAN', 'uri=' + uri.toString(), 'minEventTime=' + (minEventTime > 0 ? new Date(minEventTime).toISOString() : 'ALL'))

      const DocumentFile = plus.android.importClass('androidx.documentfile.provider.DocumentFile')
      log('SCAN', 'DocumentFile class imported')

      const root = DocumentFile.fromTreeUri(main, uri)
      if (!root) {
        log('SCAN', 'fromTreeUri returned null')
        resolve({ events: [] })
        return
      }

      const rootName = root.getName()
      log('SCAN', 'root name=' + rootName)

      const testFiles = safeListFiles(root, 'SCAN-ROOT')
      if (!testFiles) {
        log('SCAN', 'root listFiles failed or empty, permission issue?')
        resolve({ events: [] })
        return
      }

      if (rootName === 'TeslaCam') {
        log('SCAN', 'detected TeslaCam root')
        scanTeslaCamRoot(root, _minEventTime)
      } else if (CLIP_TYPES[rootName]) {
        log('SCAN', 'detected clip dir', rootName)
        scanClipDir(root, CLIP_TYPES[rootName], _minEventTime)
      } else {
        log('SCAN', 'unknown dir, searching subdirs')
        scanUnknownDir(root, _minEventTime)
      }

      const events = Object.values(_eventMap)
      log('SCAN', 'total events=' + events.length)

      events.forEach(evt => {
        const cameraCount = Object.keys(evt.videos).length
        log('EVENT', evt.eventId, 'cameras=' + cameraCount)
      })

      resolve({ events })
    } catch (e) {
      log('SCAN', 'FAILED', e.message || e)
      resolve({ events: [] })
    }
  })
}

function scanTeslaCamRoot(root, minEventTime = 0) {
  const subDirs = safeListFiles(root, 'TESLACAM')
  if (!subDirs) return
  for (let i = 0; i < subDirs.length; i++) {
    const dir = subDirs[i]
    try {
      if (dir.isDirectory()) {
        const name = dir.getName()
        log('TESLACAM', 'subdir=' + name)
        if (CLIP_TYPES[name]) {
          scanClipDir(dir, CLIP_TYPES[name], minEventTime)
        }
      }
    } catch (e) {
      log('TESLACAM', 'subdir access error', e.message || e)
    }
  }
}

function scanUnknownDir(root, minEventTime = 0) {
  const subDirs = safeListFiles(root, 'UNKNOWN')
  if (!subDirs) return
  for (let i = 0; i < subDirs.length; i++) {
    const dir = subDirs[i]
    try {
      if (dir.isDirectory()) {
        const name = dir.getName()
        log('UNKNOWN', 'subdir=' + name)
        if (name === 'TeslaCam') {
          scanTeslaCamRoot(dir, minEventTime)
        } else if (CLIP_TYPES[name]) {
          scanClipDir(dir, CLIP_TYPES[name], minEventTime)
        }
      }
    } catch (e) {
      log('UNKNOWN', 'subdir access error', e.message || e)
    }
  }
}

function scanClipDir(clipDir, eventType, minEventTime = 0) {
  const entries = safeListFiles(clipDir, 'CLIP-' + eventType)
  if (!entries) return

  let skippedCount = 0

  for (let i = 0; i < entries.length; i++) {
    const entry = entries[i]
    try {
      const name = entry.getName()
      const isDir = entry.isDirectory()

      if (isDir) {
        const eventTime = parseEventIdTime(name)
        if (eventTime && !isInRange(eventTime.getTime(), minEventTime)) {
          skippedCount++
          continue
        }
        scanEventFolder(entry, eventType, minEventTime)
        continue
      }

      if (eventType === 'recent') {
        if (minEventTime > 0) {
          const fileTime = parseEventIdTime(name)
          if (fileTime && !isInRange(fileTime.getTime(), minEventTime)) {
            skippedCount++
            continue
          }
        }
        scanSingleVideo(entry, eventType, minEventTime)
      }
    } catch (e) {
      log('CLIP', 'entry access error', e.message || e)
    }
  }

  if (skippedCount > 0) {
    log('CLIP-' + eventType, 'skipped', skippedCount, 'entries out of time range')
  }
}

function scanEventFolder(dir, eventType, minEventTime = 0) {
  const files = safeListFiles(dir, 'EVENT-' + eventType)
  if (!files) return

  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    try {
      if (!file.isDirectory()) {
        scanSingleVideo(file, eventType, minEventTime)
      }
    } catch (e) {
      log('EVENT-FOLDER', 'file access error', e.message || e)
    }
  }
}

function scanSingleVideo(file, eventType, minEventTime = 0) {
  try {
    const name = file.getName()
    if (!name || !name.toLowerCase().endsWith('.mp4')) {
      return
    }

    const camera = parseCamera(name)
    if (!camera) return

    const match = name.match(/^(\d{4}-\d{2}-\d{2})_(\d{2}-\d{2}-\d{2})/)
    if (!match) return

    const timeKey = match[1] + '_' + match[2]
    const eventId = eventType + '_' + timeKey
    const eventTime = parseEventTime(timeKey)
    if (!eventTime) return

    if (minEventTime > 0 && eventTime.getTime() < minEventTime) return

    const uriObj = file.getUri()
    const uriStr = uriObj ? String(uriObj.toString()) : ''
    const fileSize = Number(file.length()) || 0

    if (!_eventMap[eventId]) {
      _eventMap[eventId] = {
        eventId: eventId,
        eventType: String(eventType),
        eventTime: Number(eventTime.getTime()),
        videos: {
          front: null,
          back: null,
          left_repeater: null,
          right_repeater: null
        }
      }
      log('EVENT', 'created', eventId)
    }

    _eventMap[eventId].videos[camera] = {
      uriString: uriStr,
      fileName: String(name),
      size: fileSize,
      camera: String(camera)
    }

    log('VIDEO', 'ADDED', camera, 'to', eventId)
  } catch (e) {
    log('VIDEO', 'scanSingleVideo error', e.message || e)
  }
}

function enqueueCopy(taskFn) {
  const prev = _copyQueue
  _copyQueue = prev.then(async () => {
    while (_copyQueueActive) {
      await new Promise(r => setTimeout(r, 50))
    }
    _copyQueueActive = true
    try {
      return await taskFn()
    } finally {
      _copyQueueActive = false
    }
  })
  return _copyQueue
}

function doCopyFile(video, videoDir) {
  return new Promise((resolve) => {
    try {
      if (!video || !video.uriString) { resolve(null); return }

      const main = getMainActivity()
      const Uri = plus.android.importClass('android.net.Uri')
      const File = plus.android.importClass('java.io.File')
      const uri = Uri.parse(video.uriString)
      const localPath = videoDir + '/' + video.fileName
      const destFile = new File(localPath)

      if (plus.android.invoke(destFile, 'exists')) {
        const sz = Number(plus.android.invoke(destFile, 'length'))
        if (sz > 1024) {
          log('QUEUE', 'skip exists', video.fileName)
          resolve({ camera: video.camera, path: 'file://' + localPath, size: sz })
          return
        }
        plus.android.invoke(destFile, 'delete')
      }

      const parent = plus.android.invoke(destFile, 'getParentFile')
      if (parent) plus.android.invoke(parent, 'mkdirs')

      log('QUEUE', 'start copy', video.fileName, 'size=' + (video.size || '?'))
      const resolver = plus.android.invoke(main, 'getContentResolver')
      const inputStream = plus.android.invoke(resolver, 'openInputStream', uri)
      if (!inputStream) { resolve(null); return }

      const FileOutputStream = plus.android.importClass('java.io.FileOutputStream')
      const fos = new FileOutputStream(destFile)

      try {
        plus.android.invoke(inputStream, 'transferTo', fos)
        plus.android.invoke(fos, 'flush')
        plus.android.invoke(fos, 'close')
        plus.android.invoke(inputStream, 'close')
        const finalSize = Number(plus.android.invoke(destFile, 'length'))
        log('QUEUE', 'done via transferTo', video.fileName, finalSize + ' bytes')
        resolve({ camera: video.camera, path: 'file://' + localPath, size: finalSize })
        return
      } catch (transferErr) {
        log('QUEUE', 'transferTo fail, fallback chunked', transferErr.message || transferErr)
        try { plus.android.invoke(fos, 'close') } catch (ex) {}
        try { plus.android.invoke(inputStream, 'close') } catch (ex) {}
      }

      const inputStream2 = plus.android.invoke(resolver, 'openInputStream', uri)
      if (!inputStream2) { resolve(null); return }
      const fos2 = new FileOutputStream(destFile)

      let byteArr = null
      try {
        const Byte = plus.android.importClass('java.lang.Byte')
        byteArr = plus.android.invoke('java.lang.reflect.Array', 'newInstance', Byte.TYPE, 65536)
      } catch (arrErr) {
        log('QUEUE', 'Array.newInstance fail', arrErr.message || arrErr)
      }

      if (!byteArr) {
        try {
          byteArr = plus.android.newObject('[B', 65536)
        } catch (newObjErr) {
          log('QUEUE', 'newObject fail, abort', newObjErr.message || newObjErr)
          try { plus.android.invoke(fos2, 'close') } catch (ex) {}
          try { plus.android.invoke(inputStream2, 'close') } catch (ex) {}
          resolve(null)
          return
        }
      }

      let totalRead = 0
      const CHUNKS_PER_YIELD = 16

      const copyChunk = () => {
        try {
          for (let i = 0; i < CHUNKS_PER_YIELD; i++) {
            const readLen = plus.android.invoke(inputStream2, 'read', byteArr)
            if (readLen === -1) {
              plus.android.invoke(fos2, 'flush')
              plus.android.invoke(fos2, 'close')
              plus.android.invoke(inputStream2, 'close')
              const finalSize = Number(plus.android.invoke(destFile, 'length'))
              log('QUEUE', 'done via chunked', video.fileName, finalSize + ' bytes')
              resolve({ camera: video.camera, path: 'file://' + localPath, size: finalSize })
              return
            }
            plus.android.invoke(fos2, 'write', byteArr, 0, readLen)
            totalRead += readLen
          }
          setTimeout(copyChunk, 0)
        } catch (e) {
          log('QUEUE', 'chunk error', video.fileName, e.message || e)
          try { plus.android.invoke(fos2, 'close') } catch (ex) {}
          try { plus.android.invoke(inputStream2, 'close') } catch (ex) {}
          resolve(null)
        }
      }

      copyChunk()
    } catch (e) {
      log('QUEUE', 'ERROR', video.fileName, e.message || e)
      resolve(null)
    }
  })
}

function importSingleVideo(video, videoDir) {
  return enqueueCopy(() => doCopyFile(video, videoDir))
}

export async function importEvent(event, targetDir) {
  if (!isApp()) return []

  log('IMPORT', 'starting event', event.eventId)
  const basePath = getLocalBasePath()
  const date = new Date(event.eventTime)
  const dateStr = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  const eventType = event.eventType || 'recent'
  const videoDir = basePath + '/dashcam/videos/' + eventType + '/' + dateStr

  const File = plus.android.importClass('java.io.File')
  const dirFile = new File(videoDir)
  if (!plus.android.invoke(dirFile, 'exists')) {
    plus.android.invoke(dirFile, 'mkdirs')
  }
  log('IMPORT', 'dir ready', videoDir, 'type=' + eventType)

  const cameras = Object.keys(event.videos).filter(c => event.videos[c])
  const results = []

  for (const cam of cameras) {
    const v = event.videos[cam]
    log('IMPORT', 'queue', cam, v.fileName)
    const r = await importSingleVideo(v, videoDir)
    if (r) {
      log('IMPORT', cam, 'done, next file...')
    } else {
      log('IMPORT', cam, 'failed, skip')
    }
    if (r) results.push(r)
    await new Promise(r => setTimeout(r, 100))
  }

  log('IMPORT', 'event complete', event.eventId, results.length + '/' + cameras.length + ' videos')
  return results
}

export function generateThumbnail(videoPath, targetDir) {
  if (!isApp()) return Promise.resolve(null)
  return new Promise((resolve) => {
    try {
      const basePath = getLocalBasePath()
      const thumbDir = basePath + '/dashcam/thumbs'
      const realPath = videoPath.replace(/^file:\/\//, '')
      const thumbName = realPath.split('/').pop().replace(/\.[^.]+$/i, '.jpg')
      const thumbPath = thumbDir + '/' + thumbName

      const File = plus.android.importClass('java.io.File')
      const dirFile = new File(thumbDir)
      if (!plus.android.invoke(dirFile, 'exists')) {
        plus.android.invoke(dirFile, 'mkdirs')
      }

      const thumbFile = new File(thumbPath)
      if (plus.android.invoke(thumbFile, 'exists')) {
        resolve('file://' + thumbPath)
        return
      }

      const MediaMetadataRetriever = plus.android.importClass('android.media.MediaMetadataRetriever')
      const retriever = new MediaMetadataRetriever()
      plus.android.invoke(retriever, 'setDataSource', realPath)
      const bitmap = plus.android.invoke(retriever, 'getFrameAtTime', 500000)
      if (!bitmap) {
        plus.android.invoke(retriever, 'release')
        resolve(null)
        return
      }

      const FileOutputStream = plus.android.importClass('java.io.FileOutputStream')
      const CompressFormat = plus.android.importClass('android.graphics.Bitmap$CompressFormat')
      const out = new FileOutputStream(thumbFile)
      plus.android.invoke(bitmap, 'compress', CompressFormat.JPEG, 70, out)
      plus.android.invoke(out, 'flush')
      plus.android.invoke(out, 'close')
      plus.android.invoke(retriever, 'release')
      resolve('file://' + thumbPath)
    } catch (e) {
      resolve(null)
    }
  })
}

export function deleteImportedEvent(eventId, localDir) {
  if (!isApp()) return Promise.resolve()
  return new Promise((resolve) => {
    try {
      const basePath = getLocalBasePath()
      const File = plus.android.importClass('java.io.File')
      const types = ['recent', 'saved', 'sentry']
      for (const type of types) {
        const videoDir = new File(basePath + '/dashcam/videos/' + type)
        if (plus.android.invoke(videoDir, 'exists') && plus.android.invoke(videoDir, 'isDirectory')) {
          deleteFilesContaining(videoDir, eventId)
        }
      }
      const oldVideoDir = new File(basePath + '/dashcam/videos')
      if (plus.android.invoke(oldVideoDir, 'exists') && plus.android.invoke(oldVideoDir, 'isDirectory')) {
        deleteFilesContaining(oldVideoDir, eventId)
      }
      const thumbDir = new File(basePath + '/dashcam/thumbs')
      if (plus.android.invoke(thumbDir, 'exists') && plus.android.invoke(thumbDir, 'isDirectory')) {
        deleteFilesContaining(thumbDir, eventId)
      }
      resolve()
    } catch (e) {
      resolve()
    }
  })
}

function deleteFilesContaining(dirFile, pattern) {
  const files = plus.android.invoke(dirFile, 'listFiles')
  if (!files) return
  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    if (plus.android.invoke(file, 'isDirectory')) {
      deleteFilesContaining(file, pattern)
    } else if (plus.android.invoke(file, 'getName').includes(pattern)) {
      plus.android.invoke(file, 'delete')
    }
  }
}

export function waitForPlus() {
  return new Promise(resolve => {
    if (typeof plus !== 'undefined') return resolve()
    document.addEventListener('plusready', resolve, { once: true })
  })
}

export function scanLocalVideos() {
  if (!isApp()) {
    console.warn('[Dashcam:LOCAL] only available on Android App')
    return Promise.resolve([])
  }

  return new Promise((resolve) => {
    try {
      const basePath = getLocalBasePath()
      const videoBaseDir = basePath + '/dashcam/videos'
      log('LOCAL', 'scanning', videoBaseDir)

      const File = plus.android.importClass('java.io.File')
      const baseDir = new File(videoBaseDir)

      if (!plus.android.invoke(baseDir, 'exists')) {
        log('LOCAL', 'directory not found', videoBaseDir)
        resolve([])
        return
      }

      if (!plus.android.invoke(baseDir, 'isDirectory')) {
        log('LOCAL', 'not a directory', videoBaseDir)
        resolve([])
        return
      }

      const result = []
      const topDirs = safeListFiles(baseDir, 'LOCAL')
      if (!topDirs || topDirs.length === 0) {
        log('LOCAL', 'no directories found')
        resolve([])
        return
      }

      const VALID_TYPES = ['recent', 'saved', 'sentry']

      for (let i = 0; i < topDirs.length; i++) {
        const topDir = topDirs[i]
        try {
          if (!plus.android.invoke(topDir, 'isDirectory')) continue
          const topName = String(plus.android.invoke(topDir, 'getName'))

          if (VALID_TYPES.includes(topName)) {
            const dateDirs = safeListFiles(topDir, 'LOCAL-' + topName)
            if (!dateDirs) continue
            for (let j = 0; j < dateDirs.length; j++) {
              const dateDir = dateDirs[j]
              try {
                if (!plus.android.invoke(dateDir, 'isDirectory')) continue
                const dateName = String(plus.android.invoke(dateDir, 'getName'))
                scanDateDir(dateDir, dateName, topName, result)
              } catch (e) {
                log('LOCAL', 'date dir access error', e.message || e)
              }
            }
          } else if (/^\d{4}-\d{2}-\d{2}$/.test(topName)) {
            scanDateDir(topDir, topName, 'recent', result)
          }
        } catch (e) {
          log('LOCAL', 'top dir access error', e.message || e)
        }
      }

      log('LOCAL', 'total found', result.length, 'videos')
      resolve(result)
    } catch (e) {
      log('LOCAL', 'scan failed', e.message || e)
      resolve([])
    }
  })
}

function scanDateDir(dateDir, dateName, eventType, result) {
  const files = safeListFiles(dateDir, 'LOCAL-' + dateName)
  if (!files) return

  for (let j = 0; j < files.length; j++) {
    const file = files[j]
    try {
      if (plus.android.invoke(file, 'isDirectory')) continue
      const name = String(plus.android.invoke(file, 'getName'))
      if (!name.toLowerCase().endsWith('.mp4')) continue

      const absolutePath = String(plus.android.invoke(file, 'getAbsolutePath'))
      const size = Number(plus.android.invoke(file, 'length')) || 0
      const camera = parseCamera(name)

      let videoUrl = 'file://' + absolutePath
      try {
        const converted = plus.io.convertLocalFileSystemURL(absolutePath)
        if (converted) videoUrl = converted
      } catch (e) {
        log('LOCAL', 'convertLocalFileSystemURL failed, fallback to file://', e.message || e)
      }

      result.push({
        name,
        path: videoUrl,
        rawPath: absolutePath,
        date: dateName,
        size,
        camera: camera || 'unknown',
        eventType: eventType
      })
    } catch (e) {
      log('LOCAL', 'file access error', e.message || e)
    }
  }
}