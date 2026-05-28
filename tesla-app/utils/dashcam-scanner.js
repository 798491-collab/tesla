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

function isApp() {
  return typeof plus !== 'undefined' && plus.io
}

function parseEventTime(str) {
  const d = new Date(str.replace(/_/g, ' ').replace(/-/g, (m, offset) => offset > 10 ? ':' : '-'))
  return isNaN(d.getTime()) ? null : d
}

function parseCamera(fileName) {
  for (const key of Object.keys(CAMERA_MAP)) {
    if (fileName.includes(key)) return key
  }
  return null
}

function readDir(dirPath) {
  return new Promise((resolve, reject) => {
    plus.io.resolveLocalFileSystemURL(dirPath, (entry) => {
      const reader = entry.createReader()
      reader.readEntries((entries) => resolve(entries), (err) => reject(err))
    }, (err) => reject(err))
  })
}

function copyFile(src, dst) {
  return new Promise((resolve, reject) => {
    plus.io.resolveLocalFileSystemURL(src, (srcEntry) => {
      plus.io.resolveLocalFileSystemURL(dst, (dstDir) => {
        srcEntry.copyTo(dstDir, null, () => resolve(), (err) => reject(err))
      }, (err) => reject(err))
    }, (err) => reject(err))
  })
}

function deleteFile(filePath) {
  return new Promise((resolve, reject) => {
    plus.io.resolveLocalFileSystemURL(filePath, (entry) => {
      entry.remove(() => resolve(), (err) => reject(err))
    }, (err) => reject(err))
  })
}

function ensureDir(dirPath) {
  return new Promise((resolve, reject) => {
    plus.io.resolveLocalFileSystemURL(dirPath, (entry) => {
      if (entry.isDirectory) return resolve(entry)
      reject(new Error('Not a directory'))
    }, () => {
      const parent = dirPath.substring(0, dirPath.lastIndexOf('/'))
      const name = dirPath.substring(dirPath.lastIndexOf('/') + 1)
      plus.io.resolveLocalFileSystemURL(parent, (parentEntry) => {
        parentEntry.getDirectory(name, { create: true }, (dirEntry) => resolve(dirEntry), (err) => reject(err))
      }, (err) => reject(err))
    })
  })
}

export function selectTeslaCamDir() {
  if (!isApp()) {
    console.warn('File API only available on App')
    return Promise.resolve(null)
  }
  return new Promise((resolve) => {
    if (plus.io.chooseFile) {
      plus.io.chooseFile({ type: 'directory' }, (path) => resolve(path), () => resolve(null))
    } else {
      uni.chooseFile({ type: 'folder' }).then(res => resolve(res.tempFilePaths[0] || null)).catch(() => resolve(null))
    }
  })
}

export function scanTeslaCam(dirPath) {
  if (!isApp()) {
    console.warn('File API only available on App')
    return Promise.resolve({ events: [] })
  }
  const events = []
  const typeNames = Object.keys(CLIP_TYPES)

  return typeNames.reduce((promise, typeName) => {
    return promise.then(() => {
      const typePath = dirPath + '/' + typeName
      return readDir(typePath).then((dateEntries) => {
        const dateDirs = dateEntries.filter(e => e.isDirectory)
        return dateDirs.reduce((p, dateDir) => {
          return p.then(() => {
            return readDir(dateDir.toLocalURL()).then((subEntries) => {
              const entries = typeName === 'RecentClips' ? [dateDir] : subEntries.filter(e => e.isDirectory)
              return entries.reduce((pp, eventDir) => {
                return pp.then(() => {
                  return readDir(eventDir.toLocalURL()).then((fileEntries) => {
                    fileEntries.filter(e => !e.isDirectory && e.name.endsWith('.mp4')).forEach(file => {
                      const camera = parseCamera(file.name)
                      if (!camera) return
                      const nameBase = file.name.replace('.mp4', '')
                      const eventTime = parseEventTime(nameBase.split('_').slice(0, 2).join('_'))
                      if (!eventTime) return
                      events.push({
                        eventType: CLIP_TYPES[typeName],
                        eventTime: eventTime.getTime(),
                        camera,
                        filePath: file.toLocalURL(),
                        fileName: file.name
                      })
                    })
                  })
                })
              }, Promise.resolve())
            })
          })
        }, Promise.resolve())
      }).catch(() => {})
    })
  }, Promise.resolve()).then(() => ({ events }))
}

export function importEvent(event, targetDir) {
  if (!isApp()) {
    console.warn('File API only available on App')
    return Promise.resolve([])
  }
  const date = new Date(event.eventTime)
  const dateStr = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  const videoDir = targetDir + '/dashcam/videos/' + dateStr
  return ensureDir(videoDir).then(() => {
    return copyFile(event.filePath, videoDir)
  }).then(() => {
    const localPath = videoDir + '/' + event.fileName
    return [localPath]
  })
}

export function generateThumbnail(videoPath, targetDir) {
  if (!isApp()) {
    console.warn('File API only available on App')
    return Promise.resolve(null)
  }
  const thumbDir = targetDir + '/dashcam/thumbs'
  const thumbName = videoPath.split('/').pop().replace('.mp4', '.jpg')
  const thumbPath = thumbDir + '/' + thumbName

  return ensureDir(thumbDir).then(() => {
    return new Promise((resolve, reject) => {
      if (plus.zip && plus.zip.compressVideo) {
        plus.zip.compressVideo(videoPath, thumbPath, () => resolve(thumbPath), (err) => reject(err))
      } else {
        const video = document.createElement('video')
        video.src = videoPath
        video.muted = true
        video.preload = 'auto'
        video.onloadeddata = () => {
          video.currentTime = 0.5
        }
        video.onseeked = () => {
          try {
            const canvas = document.createElement('canvas')
            canvas.width = 320
            canvas.height = 180
            const ctx = canvas.getContext('2d')
            ctx.drawImage(video, 0, 0, 320, 180)
            canvas.toBlob((blob) => {
              if (!blob) return reject(new Error('Canvas toBlob failed'))
              const reader = plus.io.FileReader()
              reader.onloadend = () => {
                const base64 = reader.result
                plus.io.resolveLocalFileSystemURL(thumbDir, (dirEntry) => {
                  dirEntry.getFile(thumbName, { create: true }, (fileEntry) => {
                    fileEntry.createWriter((writer) => {
                      writer.onwrite = () => resolve(thumbPath)
                      writer.onerror = (err) => reject(err)
                      writer.write(base64)
                    })
                  }, (err) => reject(err))
                }, (err) => reject(err))
              }
              reader.readAsDataURL(blob)
            }, 'image/jpeg', 0.7)
          } catch (e) {
            reject(e)
          }
        }
        video.onerror = (err) => reject(err)
      }
    })
  })
}

export function deleteImportedEvent(eventId, localDir) {
  if (!isApp()) {
    console.warn('File API only available on App')
    return Promise.resolve()
  }
  const tasks = []
  const videoDir = localDir + '/dashcam/videos'
  const thumbDir = localDir + '/dashcam/thumbs'

  return readDir(videoDir).then((dateEntries) => {
    return dateEntries.filter(e => e.isDirectory).reduce((promise, dateDir) => {
      return promise.then(() => {
        return readDir(dateDir.toLocalURL()).then((files) => {
          files.filter(f => !f.isDirectory && f.name.includes(eventId)).forEach(f => {
            tasks.push(deleteFile(f.toLocalURL()))
          })
        })
      })
    }, Promise.resolve())
  }).catch(() => {}).then(() => {
    return readDir(thumbDir).then((files) => {
      files.filter(f => f.name.includes(eventId)).forEach(f => {
        tasks.push(deleteFile(f.toLocalURL()))
      })
    }).catch(() => {})
  }).then(() => {
    return Promise.all(tasks)
  })
}
