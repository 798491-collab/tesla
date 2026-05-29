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

function isApp() {
  return typeof plus !== 'undefined' && plus.android
}

function log(tag, ...args) {
  console.log(`[Dashcam:${tag}]`, ...args)
}

function parseEventTime(str) {
  const d = new Date(str.replace(/_/g, ' ').replace(/-/g, (m, offset) => offset > 10 ? ':' : '-'))
  return isNaN(d.getTime()) ? null : d
}

function parseCamera(fileName) {
  const lower = fileName.toLowerCase()
  for (const key of Object.keys(CAMERA_MAP)) {
    if (lower.includes(key)) return key
  }
  return null
}

function getMainActivity() {
  return plus.android.runtimeMainActivity()
}

function getLocalBasePath() {
  const main = getMainActivity()
  let dir = main.getExternalFilesDir(null)
  if (!dir) dir = main.getFilesDir()
  return dir.getAbsolutePath()
}

// ====================== 修复这里 ======================
function takePersistablePermission(main, uri) {
  try {
    // 关键修复：直接用数值替代 Intent.FLAG，避免类加载失败
    const FLAG_GRANT_READ_URI_PERMISSION = 1
    const FLAG_GRANT_WRITE_URI_PERMISSION = 2
    const takeFlags = FLAG_GRANT_READ_URI_PERMISSION | FLAG_GRANT_WRITE_URI_PERMISSION

    const resolver = main.getContentResolver()

    // 加固：先判断方法是否存在，不存在就跳过，不崩溃
    if (resolver.takePersistableUriPermission) {
      resolver.takePersistableUriPermission(uri, takeFlags)
      log('PERM', 'takePersistableUriPermission OK')
    } else {
      log('PERM', 'takePersistableUriPermission NOT SUPPORTED (SKIPPED)')
    }
  } catch (e) {
    log('PERM', 'takePersistableUriPermission FAILED (SAFE IGNORE)', e.message || '')
  }
}
// ======================================================

function ensureSAFListener() {
  if (_safListenerReady) return
  _safListenerReady = true

  if (typeof plus === 'undefined' || !plus.globalEvent) return

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
      _safResolve = doResolve

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

export function scanTeslaCam(treeUri) {
  if (!isApp()) {
    return Promise.resolve({ events: [] })
  }

  return new Promise((resolve) => {
    try {
      const main = getMainActivity()
      const uri = ensureUri(treeUri)
      if (!uri) {
        log('SCAN', 'uri is null')
        resolve({ events: [] })
        return
      }

      log('SCAN', 'uri=' + uri.toString())

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

      const events = []

      if (rootName === 'TeslaCam') {
        log('SCAN', 'detected TeslaCam root')
        scanTeslaCamRoot(root, events)
      } else if (CLIP_TYPES[rootName]) {
        log('SCAN', 'detected clip dir', rootName)
        scanClipDir(root, CLIP_TYPES[rootName], events)
      } else {
        log('SCAN', 'unknown dir, searching subdirs')
        scanUnknownDir(root, events)
      }

      log('SCAN', 'total events=' + events.length)
      resolve({ events })
    } catch (e) {
      log('SCAN', 'FAILED', e.message || e)
      resolve({ events: [] })
    }
  })
}

function scanTeslaCamRoot(root, events) {
  const subDirs = safeListFiles(root, 'TESLACAM')
  if (!subDirs) return
  for (let i = 0; i < subDirs.length; i++) {
    const dir = subDirs[i]
    try {
      if (dir.isDirectory()) {
        const name = dir.getName()
        log('TESLACAM', 'subdir=' + name)
        if (CLIP_TYPES[name]) {
          scanClipDir(dir, CLIP_TYPES[name], events)
        }
      }
    } catch (e) {
      log('TESLACAM', 'subdir access error', e.message || e)
    }
  }
}

function scanUnknownDir(root, events) {
  const subDirs = safeListFiles(root, 'UNKNOWN')
  if (!subDirs) return
  for (let i = 0; i < subDirs.length; i++) {
    const dir = subDirs[i]
    try {
      if (dir.isDirectory()) {
        const name = dir.getName()
        log('UNKNOWN', 'subdir=' + name)
        if (name === 'TeslaCam') {
          scanTeslaCamRoot(dir, events)
        } else if (CLIP_TYPES[name]) {
          scanClipDir(dir, CLIP_TYPES[name], events)
        }
      }
    } catch (e) {
      log('UNKNOWN', 'subdir access error', e.message || e)
    }
  }
}

function scanClipDir(clipDir, eventType, events) {
  const entries = safeListFiles(clipDir, 'CLIP-' + eventType)
  if (!entries) return
  for (let i = 0; i < entries.length; i++) {
    const entry = entries[i]
    try {
      if (entry.isDirectory()) {
        if (eventType === 'recent') {
          scanVideoFiles(entry, eventType, events)
        } else {
          const eventDirs = safeListFiles(entry, 'EVENT-' + eventType)
          if (!eventDirs) continue
          for (let j = 0; j < eventDirs.length; j++) {
            try {
              if (eventDirs[j].isDirectory()) {
                scanVideoFiles(eventDirs[j], eventType, events)
              }
            } catch (e) {
              log('EVENT', 'access error', e.message || e)
            }
          }
        }
      }
    } catch (e) {
      log('CLIP', 'entry access error', e.message || e)
    }
  }
}

function scanVideoFiles(dir, eventType, events) {
  const files = safeListFiles(dir, 'VIDEO-' + eventType)
  if (!files) return
  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    try {
      if (file.isDirectory()) {
        scanVideoFiles(file, eventType, events)
      } else {
        const name = file.getName()
        if (name && name.toLowerCase().endsWith('.mp4')) {
          const camera = parseCamera(name)
          if (!camera) continue
          const nameBase = name.replace(/\.[^.]+$/, '')
          const timePart = nameBase.split('_').slice(0, 2).join('_')
          const eventTime = parseEventTime(timePart)
          if (!eventTime) continue
          const uriObj = file.getUri()
        const uriStr = uriObj ? String(uriObj.toString()) : ''
        const fileSize = Number(file.length()) || 0
        events.push({
          eventType: String(eventType),
          eventTime: Number(eventTime.getTime()),
          camera: String(camera),
          fileName: String(name),
          uriString: uriStr,
          size: fileSize
        })
        }
      }
    } catch (e) {
      log('VIDEO', 'file access error', e.message || e)
    }
  }
}

export function importEvent(event, targetDir) {
  if (!isApp()) {
    return Promise.resolve([])
  }

  return new Promise((resolve, reject) => {
    try {
      const main = getMainActivity()
      const Uri = plus.android.importClass('android.net.Uri')
      const uri = Uri.parse(event.uriString)
      const basePath = getLocalBasePath()
      const date = new Date(event.eventTime)
      const dateStr = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
      const videoDir = basePath + '/dashcam/videos/' + dateStr

      const File = plus.android.importClass('java.io.File')
      const dirFile = new File(videoDir)
      if (!dirFile.exists()) dirFile.mkdirs()

      const localPath = videoDir + '/' + event.fileName
      const destFile = new File(localPath)
      if (destFile.exists()) {
        resolve(['file://' + localPath])
        return
      }

      const resolver = main.getContentResolver()
      const pfd = resolver.openFileDescriptor(uri, 'r')
      const FileInputStream = plus.android.importClass('java.io.FileInputStream')
      const FileOutputStream = plus.android.importClass('java.io.FileOutputStream')

      const fis = new FileInputStream(pfd.getFileDescriptor())
      const fos = new FileOutputStream(destFile)

      const inChannel = fis.getChannel()
      const outChannel = fos.getChannel()
      inChannel.transferTo(0, inChannel.size(), outChannel)

      outChannel.close()
      inChannel.close()
      fos.close()
      fis.close()
      pfd.close()

      resolve(['file://' + localPath])
    } catch (e) {
      console.error('importEvent failed', e)
      reject(e)
    }
  })
}

export function generateThumbnail(videoPath, targetDir) {
  if (!isApp()) {
    return Promise.resolve(null)
  }

  return new Promise((resolve) => {
    try {
      const basePath = getLocalBasePath()
      const thumbDir = basePath + '/dashcam/thumbs'
      const realPath = videoPath.replace(/^file:\/\//, '')
      const thumbName = realPath.split('/').pop().replace(/\.[^.]+$/i, '.jpg')
      const thumbPath = thumbDir + '/' + thumbName

      const File = plus.android.importClass('java.io.File')
      const dirFile = new File(thumbDir)
      if (!dirFile.exists()) dirFile.mkdirs()

      const thumbFile = new File(thumbPath)
      if (thumbFile.exists()) {
        resolve('file://' + thumbPath)
        return
      }

      const MediaMetadataRetriever = plus.android.importClass('android.media.MediaMetadataRetriever')
      const retriever = new MediaMetadataRetriever()
      retriever.setDataSource(realPath)
      const bitmap = retriever.getFrameAtTime(500000)

      if (!bitmap) {
        retriever.release()
        resolve(null)
        return
      }

      const FileOutputStream = plus.android.importClass('java.io.FileOutputStream')
      const CompressFormat = plus.android.importClass('android.graphics.Bitmap$CompressFormat')
      const out = new FileOutputStream(thumbFile)
      bitmap.compress(CompressFormat.JPEG, 70, out)
      out.flush()
      out.close()
      retriever.release()

      resolve('file://' + thumbPath)
    } catch (e) {
      console.error('generateThumbnail failed', e)
      resolve(null)
    }
  })
}

export function deleteImportedEvent(eventId, localDir) {
  if (!isApp()) {
    return Promise.resolve()
  }

  return new Promise((resolve) => {
    try {
      const basePath = getLocalBasePath()
      const File = plus.android.importClass('java.io.File')

      const videoDir = new File(basePath + '/dashcam/videos')
      if (videoDir.exists() && videoDir.isDirectory()) {
        deleteFilesContaining(videoDir, eventId)
      }

      const thumbDir = new File(basePath + '/dashcam/thumbs')
      if (thumbDir.exists() && thumbDir.isDirectory()) {
        deleteFilesContaining(thumbDir, eventId)
      }

      resolve()
    } catch (e) {
      console.error('deleteImportedEvent failed', e)
      resolve()
    }
  })
}

function deleteFilesContaining(dirFile, pattern) {
  const files = dirFile.listFiles()
  if (!files) return
  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    if (file.isDirectory()) {
      deleteFilesContaining(file, pattern)
    } else if (file.getName().includes(pattern)) {
      file.delete()
    }
  }
}