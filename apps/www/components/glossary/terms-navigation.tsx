"use client";
import React, { useState, useEffect, useRef, useCallback } from 'react'
import { ChevronUpIcon, ChevronDownIcon } from 'lucide-react'
import { Button } from "@/components/ui/button"
import { terms } from '@/app/glossary/data'
import { cn } from '@/lib/utils';
import Link from 'next/link';

export default function TermsNavigation() {
    const [startIndex, setStartIndex] = useState(0)
    const [isScrollingUp, setIsScrollingUp] = useState(false)
    const [isScrollingDown, setIsScrollingDown] = useState(false)
    const scrollIntervalRef = useRef<NodeJS.Timeout | null>(null)

    // there should be 11 terms visible at a time, 5 before and 5 after the current term alphabetically
    // sort the terms alphabetically
    const sortedTerms = terms.sort((a, b) => a.title.localeCompare(b.title))
    const visibleTerms = sortedTerms.slice(startIndex, startIndex + 11)
  
    const scroll = useCallback((direction: 'up' | 'down') => {
      setStartIndex(prevIndex => {
        if (direction === 'up') {
          return Math.max(0, prevIndex - 1)
        } 
          return Math.min(terms.length - 5, prevIndex + 1)
      })
    }, [terms.length])
  
    const startScrolling = useCallback((direction: 'up' | 'down') => {
      if (direction === 'up') {
        setIsScrollingUp(true)
      } else {
        setIsScrollingDown(true)
      }
      scroll(direction)
      scrollIntervalRef.current = setInterval(() => scroll(direction), 150)
    }, [scroll])
  
    const stopScrolling = useCallback(() => {
      if (scrollIntervalRef.current) {
        clearInterval(scrollIntervalRef.current)
        scrollIntervalRef.current = null
      }
      setIsScrollingUp(false)
      setIsScrollingDown(false)
    }, [])
  
    useEffect(() => {
      return () => {
        if (scrollIntervalRef.current) {
          clearInterval(scrollIntervalRef.current)
        }
      }
    }, [])
  
    return (
      <div className="w-full max-w-[15rem] space-y-4 p-4 text-white min-h-screen">
        <div className="flex flex-col">
          <Button
            variant="ghost"
            size="icon"
            onMouseDown={() => startScrolling('up')}
            onMouseUp={stopScrolling}
            onMouseLeave={stopScrolling}
            disabled={startIndex === 0}
            className="p-0 self-center mb-2 transition-colors duration-150"
          >
            <ChevronUpIcon className="w-4 h-4" />
            <span className="sr-only">Scroll up</span>
          </Button>
          <div className="space-y-2 overflow-hidden">
            {visibleTerms.map((term, index) => (
              <Link
                key={term.slug}
                href={`/glossary/${term.slug}`}
                className={cn("flex items-center px-2 py-1 h-10 rounded-md transition-all duration-300 ease-in-out", {
                    "opacity-70": isScrollingUp || isScrollingDown || index !== 2,
                    "opacity-100": !isScrollingUp && !isScrollingDown && index === 2,
                })}
              >
                <span className="text-sm font-normal">{term.title}</span>
              </Link>
            ))}
          </div>
          <Button
            variant="ghost"
            size="icon"
            onMouseDown={() => startScrolling('down')}
            onMouseUp={stopScrolling}
            onMouseLeave={stopScrolling}
            disabled={startIndex >= terms.length - 5}
            className={`p-0 self-center mt-2 transition-colors duration-150`}
          >
            <ChevronDownIcon className="w-4 h-4" />
            <span className="sr-only">Scroll down</span>
          </Button>
        </div>
      </div>
    )
  }
  