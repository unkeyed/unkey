'use client'

import { useRef } from 'react'
import Image from 'next/image'
import {
  motion,
  useMotionTemplate,
  useScroll,
  useTransform,
} from 'framer-motion'

const MotionImage = motion(Image)

export function GrayscaleTransitionImage(props) {
  let ref = useRef()
  let { scrollYProgress } = useScroll({
    target: ref,
    offset: ['start 65%', 'end 35%'],
  })
  let grayscale = useTransform(scrollYProgress, [0, 0.5, 1], [1, 0, 1])
  let filter = useMotionTemplate`grayscale(${grayscale})`

  return (
    <div ref={ref} className="group relative">
      <MotionImage alt="" style={{ filter }} {...props} />
      <div
        className="pointer-events-none absolute left-0 top-0 w-full opacity-0 transition duration-300 group-hover:opacity-100"
        aria-hidden="true"
      >
        <Image alt="" {...props} />
      </div>
    </div>
  )
}
