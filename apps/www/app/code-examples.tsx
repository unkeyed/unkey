"use client";
import { Editor } from "@/components/analytics/analytics-bento";
import { PrimaryButton, SecondaryButton } from "@/components/button";
import { SectionTitle } from "@/components/section";
import { MeteorLines } from "@/components/ui/meteorLines";
import { cn } from "@/lib/utils";
import * as TabsPrimitive from "@radix-ui/react-tabs";
import { ChevronRight } from "lucide-react";
import Link from "next/link";
import type { PrismTheme } from "prism-react-renderer";
import React, { useEffect } from "react";
import { useState } from "react";
const Tabs = TabsPrimitive.Root;

const editorTheme = {
  plain: {
    color: "#F8F8F2",
    backgroundColor: "#282A36",
  },
  styles: [
    {
      types: ["keyword"],
      style: {
        color: "#9D72FF",
      },
    },
    {
      types: ["function"],
      style: {
        color: "#FB3186",
      },
    },
    {
      types: ["string"],
      style: {
        color: "#3CEEAE",
      },
    },
    {
      types: ["string-property"],
      style: {
        color: "#9D72FF",
      },
    },
    {
      types: ["number"],
      style: {
        color: "#FB3186",
      },
    },
    {
      types: ["comment"],
      style: {
        color: "#4D4D4D",
      },
    },
  ],
} satisfies PrismTheme;

type IconProps = {
  active: boolean;
};

const JavaIcon: React.FC<IconProps> = ({ active }) => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
    <g opacity={active ? 1 : 0.3}>
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M8.02611 17.8886C7.85846 17.7748 7.69538 17.6535 7.53739 17.5251C7.50312 17.4972 7.46909 17.4691 7.4353 17.4405L7.17192 17.2061C7.08767 17.1283 7.00541 17.0486 6.92519 16.9669C6.84616 16.8865 6.76911 16.8043 6.69411 16.7203C6.21543 16.1844 5.81995 15.5778 5.52221 14.9207C5.13048 14.0561 4.91594 13.1216 4.89129 12.1726C4.86663 11.2235 5.03237 10.2791 5.37868 9.39533C5.725 8.51153 6.24486 7.70625 6.90751 7.02714C7.57015 6.34803 8.36212 5.80889 9.23656 5.44161C10.111 5.07434 11.0501 4.8864 11.9984 4.88891C12.9466 4.89142 13.8848 5.08434 14.7572 5.45623C15.3549 5.71098 15.9134 6.04616 16.4178 6.45143C16.6497 6.63782 16.8703 6.83903 17.0779 7.05407C17.1382 6.97357 17.1964 6.89165 17.2527 6.80841C17.3666 6.63979 17.4721 6.46571 17.5688 6.28681C17.6682 6.10305 17.7583 5.9142 17.8388 5.72097C17.975 6.137 18.1031 6.54279 18.2216 6.93812C18.8792 9.13202 19.2414 11.004 19.0679 12.5183C18.9759 13.8241 18.5247 15.0789 17.7643 16.1438C17.0039 17.2087 15.9639 18.042 14.7596 18.5516C13.5552 19.0612 12.2335 19.2272 10.9408 19.0311C9.89323 18.8722 8.89784 18.4805 8.02611 17.8886ZM19.9532 12.6001C19.8466 14.0624 19.3396 15.4672 18.4877 16.6603C17.6319 17.8587 16.4615 18.7967 15.106 19.3703C13.7504 19.9439 12.2626 20.1307 10.8075 19.9099C9.35233 19.6892 7.98672 19.0695 6.86184 18.1197L6.853 18.1122L6.57487 17.8647L6.56885 17.8591C5.7845 17.1348 5.15331 16.2603 4.71255 15.2875C4.2718 14.3148 4.03043 13.2634 4.0027 12.1957C3.97496 11.1279 4.16142 10.0654 4.55106 9.07103C4.94071 8.07665 5.52564 7.17054 6.2713 6.40636C7.01697 5.64217 7.90822 5.03542 8.89234 4.62208C9.87647 4.20873 10.9335 3.9972 12.0007 4.00003C13.068 4.00285 14.1239 4.21998 15.1058 4.63853C15.7413 4.9094 16.88 5.68372 16.88 5.68372C16.88 5.68372 17.7524 4.53779 18.2216 3.99999L18.6835 5.44426C19.5714 8.15493 20.1764 10.6022 19.9532 12.6001Z"
        fill={active ? "url(#paint0_linear_574_1420)" : "white"}
      />
      <path
        d="M17.8388 5.72097C17.6413 6.19518 17.3857 6.64298 17.0779 7.05407C16.4188 6.37146 15.6297 5.82813 14.7572 5.45623C13.8848 5.08434 12.9466 4.89142 11.9984 4.88891C11.0501 4.8864 10.111 5.07434 9.23656 5.44161C8.36212 5.80889 7.57015 6.34803 6.90751 7.02714C6.24486 7.70625 5.725 8.51153 5.37868 9.39533C5.03237 10.2791 4.86663 11.2235 4.89129 12.1726C4.91594 13.1216 5.13048 14.0561 5.52221 14.9207C5.91395 15.7853 6.47491 16.5624 7.17192 17.2061L7.4353 17.4405C8.43478 18.2845 9.64807 18.835 10.9408 19.0311C12.2335 19.2272 13.5552 19.0612 14.7596 18.5516C15.9639 18.042 17.0039 17.2087 17.7643 16.1438C18.5247 15.0789 18.9759 13.8241 19.0679 12.5183C19.2727 10.7311 18.7313 8.44577 17.8388 5.72097ZM8.19617 17.2647C8.12094 17.3573 8.01999 17.4254 7.90608 17.4606C7.79217 17.4957 7.67042 17.4963 7.5562 17.4621C7.44199 17.428 7.34044 17.3608 7.26439 17.2689C7.18834 17.177 7.1412 17.0646 7.12894 16.9459C7.11667 16.8272 7.13983 16.7075 7.19548 16.602C7.25113 16.4965 7.33678 16.4098 7.4416 16.353C7.54642 16.2962 7.66571 16.2718 7.78439 16.2828C7.90307 16.2939 8.01582 16.3399 8.10838 16.4151C8.22972 16.518 8.30662 16.6639 8.32298 16.8222C8.33935 16.9806 8.29391 17.1391 8.19617 17.2647ZM17.7949 15.1406C16.0536 17.4698 12.3078 16.6788 9.92277 16.7959C9.92277 16.7959 9.49843 16.8252 9.0741 16.8838C9.0741 16.8838 9.23505 16.8106 9.4399 16.7374C11.1226 16.1514 11.9127 16.0342 12.937 15.5068C14.8538 14.5253 16.7706 12.3718 17.151 10.1451C16.4194 12.2839 14.1954 14.1297 12.1761 14.8769C10.7861 15.3896 8.28397 15.8877 8.28397 15.8877L8.18154 15.8291C6.48421 14.9941 6.42568 11.3024 9.5277 10.1158C10.8885 9.58842 12.1761 9.88141 13.654 9.52982C15.2196 9.16359 17.034 7.99163 17.7656 6.45344C18.585 8.9292 19.58 12.7674 17.7949 15.1406Z"
        fill={active ? "url(#paint0_linear_574_1420)" : "white"}
      />
    </g>
  </svg>
);

const ElixirIcon: React.FC<IconProps> = ({ active }) => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
    <path
      opacity={active ? 1 : 0.3}
      fillRule="evenodd"
      clipRule="evenodd"
      d="M13.0662 3.80724L13.0932 3L12.2819 3.26983C10.8533 3.74493 9.59014 5.08057 8.44142 7.06638C7.59444 8.53059 6.58016 10.3729 6.17808 12.2861C5.77107 14.2228 5.98352 16.2776 7.64139 18.0908C8.4054 18.9263 9.43595 19.6094 10.7112 19.878C11.9922 20.1478 13.4633 19.9871 15.0783 19.2252C16.4788 18.5645 17.3384 17.3062 17.7336 16.0465C18.1243 14.8013 18.0987 13.4326 17.5674 12.4912C16.6865 10.9301 15.7955 9.92149 15.0739 9.10465L14.952 8.96654C14.2055 8.11946 13.7155 7.51862 13.506 6.67094C13.1826 5.36299 13.0459 4.41825 13.0662 3.80724ZM9.47529 7.59289C10.3192 6.13401 11.1458 5.19256 11.9328 4.677C12.0012 5.30498 12.1463 6.05546 12.3601 6.92033C12.6395 8.0505 13.3069 8.83191 14.0455 9.66992L14.1626 9.80261C14.8796 10.6145 15.7045 11.5486 16.5279 13.0076C16.869 13.6123 16.9418 14.6618 16.6051 15.7348C16.2729 16.7933 15.5795 17.7533 14.5498 18.2391C13.1397 18.9043 11.9457 19.0067 10.9688 18.801C9.98626 18.5942 9.16622 18.0633 8.53417 17.3721C7.17433 15.8849 6.97368 14.2055 7.33219 12.4997C7.69561 10.7704 8.62549 9.06196 9.47529 7.59289ZM8.03541 14.3836C8.2029 15.1308 8.56477 15.8565 9.18027 16.5296C9.97991 17.4041 11.1649 18.0623 12.6712 17.9957L12.6159 16.8932C11.5436 16.9406 10.6877 16.4832 10.073 15.811C9.58713 15.2795 9.31357 14.723 9.18656 14.1565L8.03541 14.3836Z"
      fill={active ? "url(#paint0_linear_574_1420)" : "white"}
    />
  </svg>
);

const RustIcon: React.FC<IconProps> = ({ active }) => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
    <g opacity={active ? 1 : 0.3}>
      <path
        d="M15.7274 16.584C15.2794 16.584 14.8954 16.528 14.5754 16.416C14.2634 16.296 14.0034 16.14 13.7954 15.948C13.5954 15.748 13.4474 15.516 13.3514 15.252C13.2634 14.988 13.2194 14.708 13.2194 14.412V13.752C13.2194 13.472 13.2034 13.244 13.1714 13.068C13.1474 12.892 13.0994 12.756 13.0274 12.66C12.9634 12.564 12.8674 12.5 12.7394 12.468C12.6194 12.436 12.4634 12.42 12.2714 12.42H11.3234V14.688C11.3234 15.056 11.7674 15.24 12.6554 15.24V16.5H7.21939V15.24C7.93139 15.24 8.28739 15.056 8.28739 14.688V9.396C8.28739 9.028 7.93139 8.844 7.21939 8.844V7.584H13.4714C13.8634 7.584 14.2474 7.612 14.6234 7.668C14.9994 7.716 15.3314 7.82 15.6194 7.98C15.9154 8.14 16.1514 8.368 16.3274 8.664C16.5034 8.952 16.5914 9.336 16.5914 9.816C16.5914 10.128 16.5434 10.404 16.4474 10.644C16.3514 10.876 16.2234 11.076 16.0634 11.244C15.9114 11.412 15.7354 11.548 15.5354 11.652C15.3354 11.756 15.1274 11.832 14.9114 11.88C15.3434 11.992 15.6754 12.2 15.9074 12.504C16.1394 12.808 16.2554 13.268 16.2554 13.884V14.712C16.2554 14.888 16.2914 15.02 16.3634 15.108C16.4354 15.188 16.5274 15.228 16.6394 15.228C16.7594 15.228 16.8554 15.188 16.9274 15.108C16.9994 15.02 17.0354 14.888 17.0354 14.712V14.136H17.9114V14.688C17.9114 14.952 17.8674 15.2 17.7794 15.432C17.6994 15.664 17.5674 15.864 17.3834 16.032C17.2074 16.2 16.9834 16.336 16.7114 16.44C16.4394 16.536 16.1114 16.584 15.7274 16.584ZM11.3234 11.22H12.4154C12.6314 11.22 12.8074 11.2 12.9434 11.16C13.0874 11.12 13.1994 11.056 13.2794 10.968C13.3674 10.872 13.4274 10.748 13.4594 10.596C13.4914 10.436 13.5074 10.24 13.5074 10.008C13.5074 9.736 13.4314 9.524 13.2794 9.372C13.1274 9.22 12.8394 9.144 12.4154 9.144H11.3234V11.22Z"
        fill={active ? "url(#paint0_linear_574_1420)" : "white"}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M12 20C16.4183 20 20 16.4183 20 12C20 7.58172 16.4183 4 12 4C7.58172 4 4 7.58172 4 12C4 16.4183 7.58172 20 12 20ZM12 21C16.9706 21 21 16.9706 21 12C21 7.02944 16.9706 3 12 3C7.02944 3 3 7.02944 3 12C3 16.9706 7.02944 21 12 21Z"
        fill={active ? "url(#paint0_linear_574_1420)" : "white"}
      />
    </g>
  </svg>
);

const CurlIcon: React.FC<IconProps> = ({ active }) => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
    <g opacity={active ? 1 : 0.3}>
      <path
        d="M18.8202 6.91608C18.6492 6.91608 18.4852 6.84502 18.3643 6.71854C18.2435 6.59205 18.1755 6.4205 18.1755 6.24163C18.1755 6.06275 18.2435 5.8912 18.3643 5.76471C18.4852 5.63823 18.6492 5.56717 18.8202 5.56717C18.9911 5.56717 19.1551 5.63823 19.276 5.76471C19.3969 5.8912 19.4648 6.06275 19.4648 6.24163C19.4648 6.4205 19.3969 6.59205 19.276 6.71854C19.1551 6.84502 18.9911 6.91608 18.8202 6.91608ZM12.4177 18.4394C12.2467 18.4394 12.0827 18.3683 11.9618 18.2418C11.841 18.1154 11.773 17.9438 11.773 17.7649C11.773 17.5861 11.841 17.4145 11.9618 17.288C12.0827 17.1615 12.2467 17.0905 12.4177 17.0905C12.5886 17.0905 12.7526 17.1615 12.8735 17.288C12.9944 17.4145 13.0623 17.5861 13.0623 17.7649C13.0623 17.9438 12.9944 18.1154 12.8735 18.2418C12.7526 18.3683 12.5886 18.4394 12.4177 18.4394ZM18.8202 5.0072C18.5074 5.00772 18.2076 5.13794 17.9864 5.36933C17.7653 5.60071 17.6408 5.91439 17.6403 6.24163C17.6403 6.38685 17.6753 6.52292 17.7197 6.65245L12.1832 16.5796C11.648 16.6973 11.2378 17.1703 11.2378 17.7649C11.2383 18.0922 11.3628 18.4058 11.5839 18.6372C11.8051 18.8686 12.1049 18.9988 12.4177 18.9994C12.7304 18.9988 13.0302 18.8686 13.2514 18.6372C13.4725 18.4058 13.597 18.0922 13.5975 17.7649C13.5975 17.6341 13.5625 17.5033 13.5225 17.3724L19.0871 7.41195C19.6061 7.28111 20 6.82319 20 6.23443C19.9995 5.9072 19.875 5.59352 19.6539 5.36213C19.4327 5.13074 19.1329 5.00052 18.8202 5"
        fill={active ? "url(#paint0_linear_574_1420)" : "white"}
      />
      <path
        d="M13.9977 6.91607C13.8267 6.91607 13.6627 6.84501 13.5418 6.71853C13.4209 6.59204 13.353 6.42049 13.353 6.24162C13.353 6.06274 13.4209 5.89119 13.5418 5.7647C13.6627 5.63822 13.8267 5.56716 13.9977 5.56716C14.1686 5.56716 14.3326 5.63822 14.4535 5.7647C14.5744 5.89119 14.6423 6.06274 14.6423 6.24162C14.6423 6.42049 14.5744 6.59204 14.4535 6.71853C14.3326 6.84501 14.1686 6.91607 13.9977 6.91607ZM7.5889 18.44C7.41794 18.44 7.25397 18.369 7.13308 18.2425C7.01219 18.116 6.94427 17.9444 6.94427 17.7656C6.94427 17.5867 7.01219 17.4151 7.13308 17.2887C7.25397 17.1622 7.41794 17.0911 7.5889 17.0911C7.75987 17.0911 7.92383 17.1622 8.04472 17.2887C8.16561 17.4151 8.23353 17.5867 8.23353 17.7656C8.23353 17.9444 8.16561 18.116 8.04472 18.2425C7.92383 18.369 7.75987 18.44 7.5889 18.44ZM13.9977 5.00719C13.3455 5.00719 12.8178 5.55997 12.8178 6.24162C12.8178 6.38684 12.8528 6.52291 12.8972 6.65244L7.35444 16.5796C6.81923 16.6973 6.40907 17.1703 6.40907 17.7649C6.4094 18.0923 6.53379 18.4061 6.75496 18.6376C6.97613 18.8692 7.27603 18.9995 7.5889 19C7.90166 18.9995 8.20147 18.8693 8.42262 18.6379C8.64378 18.4065 8.76824 18.0928 8.76874 17.7656C8.76874 17.6347 8.73372 17.5039 8.69371 17.3731L14.2584 7.41259C14.7773 7.28175 15.1712 6.82383 15.1712 6.23507C15.1691 5.90898 15.0439 5.59697 14.8229 5.367C14.602 5.13702 14.3031 5.0077 13.9914 5.00719M5.17358 8.98065C5.34455 8.98065 5.50851 9.05171 5.6294 9.17819C5.75029 9.30468 5.81821 9.47623 5.81821 9.6551C5.81821 9.83398 5.75029 10.0055 5.6294 10.132C5.50851 10.2585 5.34455 10.3296 5.17358 10.3296C5.00262 10.3296 4.83865 10.2585 4.71776 10.132C4.59687 10.0055 4.52896 9.83398 4.52896 9.6551C4.52896 9.47623 4.59687 9.30468 4.71776 9.17819C4.83865 9.05171 5.00262 8.98065 5.17358 8.98065ZM5.17358 10.8895C5.48634 10.889 5.78615 10.7588 6.00731 10.5274C6.22846 10.296 6.35292 9.98234 6.35342 9.6551C6.35342 9.52427 6.31778 9.39343 6.27839 9.2626C6.12208 8.77197 5.69379 8.41348 5.17296 8.41348C5.0898 8.41348 5.01665 8.44619 4.93787 8.4632C4.40391 8.58095 4 9.05326 4 9.64856C4.0005 9.97579 4.12496 10.2895 4.34611 10.5209C4.56727 10.7523 4.86708 10.8825 5.17984 10.883M4.52896 13.9694C4.52896 13.7905 4.59687 13.619 4.71776 13.4925C4.83865 13.366 5.00262 13.2949 5.17358 13.2949C5.34455 13.2949 5.50851 13.366 5.6294 13.4925C5.75029 13.619 5.81821 13.7905 5.81821 13.9694C5.81821 14.1483 5.75029 14.3198 5.6294 14.4463C5.50851 14.5728 5.34455 14.6438 5.17358 14.6438C5.00262 14.6438 4.83865 14.5728 4.71776 14.4463C4.59687 14.3198 4.52896 14.1483 4.52896 13.9694ZM6.35342 13.9694C6.35342 13.8386 6.31778 13.7077 6.27839 13.5769C6.12208 13.0863 5.69441 12.7278 5.17296 12.7278C5.0898 12.7278 5.01665 12.7605 4.93787 12.7768C4.40391 12.8946 4 13.3675 4 13.9628C4.0005 14.2901 4.12496 14.6038 4.34611 14.8351C4.56727 15.0665 4.86708 15.1968 5.17984 15.1973C5.4926 15.1968 5.7924 15.0665 6.01356 14.8351C6.23471 14.6038 6.35918 14.2901 6.35967 13.9628"
        fill={active ? "url(#paint0_linear_574_1420)" : "white"}
      />
      <defs>
        <linearGradient
          id="paint0_linear_574_1420"
          x1="4.15606"
          y1="2.27462"
          x2="4.15606"
          y2="20.9494"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.4" />
          <stop offset="1" stopColor="white" />
        </linearGradient>
      </defs>
    </g>
  </svg>
);

const PythonIcon: React.FC<IconProps> = ({ active }) => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
    <path
      opacity={active ? 1 : 0.3}
      fillRule="evenodd"
      clipRule="evenodd"
      d="M8.92308 6.34188C8.92308 6.14145 9.10073 5.81724 9.69579 5.51187C10.2601 5.2223 11.0743 5.02564 12 5.02564C12.9257 5.02564 13.7399 5.2223 14.3042 5.51187C14.8993 5.81724 15.0769 6.14145 15.0769 6.34188L15.0768 8.41025V8.58119C15.0768 9.42328 15.0233 10.0118 14.9146 10.4281C14.8088 10.8329 14.6614 11.0364 14.4939 11.1606C14.313 11.2945 14.0407 11.3872 13.5869 11.4368C13.1321 11.4865 12.5656 11.4872 11.829 11.4872H11.803C11.1831 11.4872 10.623 11.4872 10.1514 11.5443C9.6692 11.6028 9.2122 11.7273 8.83772 12.0269C8.4557 12.3325 8.22386 12.7678 8.08632 13.3179C7.96911 13.7868 7.91331 14.3669 7.90039 15.0769H6.34188C6.14145 15.0769 5.81723 14.8993 5.51187 14.3042C5.22229 13.7399 5.02564 12.9257 5.02564 12C5.02564 11.0743 5.22229 10.2601 5.51187 9.6958C5.81723 9.10073 6.14145 8.92308 6.34188 8.92308H8.41026H12V7.89744H8.92308V6.34188ZM6.34188 16.1026H7.89741L7.89744 17.6582C7.89744 18.4489 8.52337 19.0393 9.22754 19.4006C9.96246 19.7777 10.9431 20 12 20C13.0569 20 14.0375 19.7777 14.7724 19.4006C15.4766 19.0393 16.1026 18.4489 16.1026 17.6582V16.1026H17.6581C18.4489 16.1026 19.0393 15.4766 19.4006 14.7724C19.7777 14.0375 20 13.0569 20 12C20 10.9431 19.7777 9.96246 19.4006 9.22754C19.0393 8.52338 18.4489 7.89744 17.6581 7.89744H16.1026V6.34189C16.1026 5.55103 15.4766 4.96071 14.7724 4.59937C14.0375 4.22223 13.0569 4 12 4C10.9431 4 9.96246 4.22223 9.22754 4.59937C8.52337 4.96071 7.89744 5.55103 7.89744 6.34188V7.89744H6.34188C5.55102 7.89744 4.96071 8.52338 4.59937 9.22754C4.22223 9.96246 4 10.9431 4 12C4 13.0569 4.22223 14.0375 4.59937 14.7724C4.96071 15.4766 5.55102 16.1026 6.34188 16.1026ZM12 16.1026H15.0769V17.6582C15.0769 17.8586 14.8993 18.1828 14.3042 18.4881C13.7399 18.7777 12.9257 18.9744 12 18.9744C11.0743 18.9744 10.2601 18.7777 9.69579 18.4881C9.10073 18.1828 8.92308 17.8586 8.92308 17.6582L8.92303 15.5897V15.4188C8.92303 14.5776 8.97644 13.9862 9.08133 13.5667C9.18418 13.1553 9.32627 12.9495 9.47843 12.8278C9.63814 12.7 9.87559 12.6109 10.2748 12.5625C10.678 12.5136 11.1772 12.5128 11.829 12.5128H11.8539C12.56 12.5128 13.1816 12.5128 13.6983 12.4564C14.222 12.3991 14.7082 12.2781 15.1043 11.9847C15.5136 11.6816 15.7615 11.2441 15.9069 10.6873C16.0301 10.2157 16.0865 9.63309 16.0995 8.92308H17.6581C17.8586 8.92308 18.1828 9.10073 18.4881 9.6958C18.7776 10.2601 18.9744 11.0743 18.9744 12C18.9744 12.9257 18.7776 13.7399 18.4881 14.3042C18.1828 14.8993 17.8586 15.0769 17.6581 15.0769H15.5898H12V16.1026ZM10.1806 7.33585C10.5353 7.33585 10.8229 7.04829 10.8229 6.69356C10.8229 6.33884 10.5353 6.05128 10.1806 6.05128C9.82592 6.05128 9.53836 6.33884 9.53836 6.69356C9.53836 7.04829 9.82592 7.33585 10.1806 7.33585ZM13.8227 17.9533C14.1781 17.9533 14.4661 17.6653 14.4661 17.3101C14.4661 16.9547 14.1781 16.6667 13.8227 16.6667C13.4674 16.6667 13.1794 16.9547 13.1794 17.3101C13.1794 17.6653 13.4674 17.9533 13.8227 17.9533Z"
      fill={active ? "url(#paint0_linear_574_1420)" : "white"}
    />
    <defs>
      <linearGradient
        id="paint0_linear_574_1420"
        x1="4.15606"
        y1="2.27462"
        x2="4.15606"
        y2="20.9494"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="white" stopOpacity="0.4" />
        <stop offset="1" stopColor="white" />
      </linearGradient>
    </defs>
  </svg>
);

const TSIcon: React.FC<IconProps> = ({ active }) => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
    <g opacity={active ? 1 : 0.3}>
      <path
        d="M12.4559 14.424C12.4559 15.432 13.2239 16.08 14.4159 16.08C15.5839 16.08 16.3439 15.408 16.3439 14.4C16.3439 13.52 15.8319 12.888 14.9199 12.648L14.2079 12.448C13.7919 12.344 13.5679 12.064 13.5679 11.696C13.5679 11.232 13.8799 10.952 14.3999 10.952C14.9359 10.952 15.2559 11.24 15.2559 11.688H16.2559C16.2559 10.704 15.5359 10.08 14.4079 10.08C13.2879 10.08 12.5759 10.72 12.5759 11.712C12.5759 12.568 13.1039 13.2 14.0079 13.44L14.7119 13.632C15.1119 13.736 15.3439 14.04 15.3439 14.432C15.3439 14.904 14.9999 15.208 14.4239 15.208C13.8399 15.208 13.4559 14.896 13.4559 14.424H12.4559Z"
        fill={active ? "url(#paint0_linear_574_1420)" : "white"}
      />
      <path
        d="M9.09904 11.064V16H10.107V11.064H11.635V10.16H7.57104V11.064H9.09904Z"
        fill={active ? "url(#paint0_linear_574_1420)" : "white"}
      />
      <path
        fillRule="evenodd"
        clipRule="evenodd"
        d="M9.27779 4C8.45652 4 7.80955 3.99999 7.28889 4.04253C6.75771 4.08593 6.31414 4.17609 5.91103 4.38148C5.25247 4.71703 4.71703 5.25247 4.38148 5.91103C4.17609 6.31414 4.08593 6.75771 4.04253 7.28889C3.99999 7.80955 4 8.45652 4 9.27779V14.7222C4 15.5435 3.99999 16.1905 4.04253 16.7111C4.08593 17.2423 4.17609 17.6859 4.38148 18.089C4.71703 18.7475 5.25247 19.283 5.91103 19.6185C6.31414 19.8239 6.75771 19.9141 7.28889 19.9575C7.80953 20 8.45649 20 9.27773 20H14.7222C15.5435 20 16.1905 20 16.7111 19.9575C17.2423 19.9141 17.6859 19.8239 18.089 19.6185C18.7475 19.283 19.283 18.7475 19.6185 18.089C19.8239 17.6859 19.9141 17.2423 19.9575 16.7111C20 16.1905 20 15.5435 20 14.7223V9.27778C20 8.45654 20 7.80953 19.9575 7.28889C19.9141 6.75771 19.8239 6.31414 19.6185 5.91103C19.283 5.25247 18.7475 4.71703 18.089 4.38148C17.6859 4.17609 17.2423 4.08593 16.7111 4.04253C16.1905 3.99999 15.5435 4 14.7222 4H9.27779ZM6.36502 5.27248C6.60366 5.15089 6.90099 5.07756 7.37032 5.03921C7.84549 5.00039 8.45167 5 9.3 5H14.7C15.5483 5 16.1545 5.00039 16.6297 5.03921C17.099 5.07756 17.3963 5.15089 17.635 5.27248C18.1054 5.51217 18.4878 5.89462 18.7275 6.36502C18.8491 6.60366 18.9224 6.90099 18.9608 7.37032C18.9996 7.84549 19 8.45167 19 9.3V14.7C19 15.5483 18.9996 16.1545 18.9608 16.6297C18.9224 17.099 18.8491 17.3963 18.7275 17.635C18.4878 18.1054 18.1054 18.4878 17.635 18.7275C17.3963 18.8491 17.099 18.9224 16.6297 18.9608C16.1545 18.9996 15.5483 19 14.7 19H9.3C8.45167 19 7.84549 18.9996 7.37032 18.9608C6.90099 18.9224 6.60366 18.8491 6.36502 18.7275C5.89462 18.4878 5.51217 18.1054 5.27248 17.635C5.15089 17.3963 5.07756 17.099 5.03921 16.6297C5.00039 16.1545 5 15.5483 5 14.7V9.3C5 8.45167 5.00039 7.84549 5.03921 7.37032C5.07756 6.90099 5.15089 6.60366 5.27248 6.36502C5.51217 5.89462 5.89462 5.51217 6.36502 5.27248Z"
        fill={active ? "url(#paint0_linear_574_1420)" : "white"}
      />
    </g>
  </svg>
);

const GoIcon: React.FC<IconProps> = ({ active }) => (
  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
    <path
      opacity={active ? 1 : 0.3}
      fillRule="evenodd"
      clipRule="evenodd"
      d="M10.5578 10.1111C10.247 10.1919 9.93194 10.2741 9.56747 10.3666L9.54556 10.3725C9.36723 10.4197 9.34879 10.4244 9.18283 10.2364C8.98432 10.015 8.83829 9.87162 8.55983 9.74107C7.72484 9.33718 6.91624 9.4544 6.16041 9.93662C5.25909 10.51 4.7951 11.3572 4.80843 12.4128C4.82176 13.4553 5.55064 14.3155 6.59802 14.4589C7.49935 14.5761 8.25489 14.2633 8.85165 13.5986C8.94014 13.4922 9.0212 13.3789 9.11142 13.253C9.14261 13.2089 9.17516 13.1636 9.20956 13.1164H6.65106C6.37256 13.1164 6.30619 12.9469 6.39922 12.7255C6.57136 12.3216 6.88987 11.6439 7.07534 11.305C7.11506 11.2269 7.20778 11.0966 7.40665 11.0966H11.6727C11.8642 10.4997 12.1753 9.93579 12.5899 9.40218C13.5576 8.15134 14.7241 7.49939 16.3015 7.22606C17.6538 6.99134 18.9263 7.12161 20.0795 7.89051C21.127 8.5944 21.7763 9.54551 21.9486 10.7966C22.174 12.5561 21.657 13.9897 20.4244 15.2147C19.5494 16.0878 18.4757 16.635 17.2428 16.8828C17.0074 16.9256 16.7723 16.9458 16.5409 16.9658C16.4203 16.9761 16.3004 16.9867 16.1825 17C14.9759 16.9739 13.8759 16.635 12.9478 15.8533C12.2952 15.2986 11.8454 14.6169 11.6222 13.8233C11.4654 14.1342 11.2789 14.4299 11.0652 14.7064C10.1109 15.9444 8.86465 16.7133 7.28748 16.9219C5.98826 17.0914 4.78177 16.8436 3.72135 16.0617C2.74034 15.3319 2.18361 14.3675 2.03782 13.1686C1.86539 11.748 2.28967 10.4711 3.16462 9.35023C4.10593 8.13829 5.35184 7.36939 6.87625 7.09578C8.12247 6.87411 9.31537 7.01745 10.3893 7.73411C11.0919 8.19023 11.5956 8.81579 11.927 9.57163C12.0065 9.68913 11.9535 9.75413 11.7944 9.7933C11.3815 9.89679 10.9694 10.0029 10.5578 10.1111ZM19.328 11.5958C19.3307 11.6394 19.3336 11.6855 19.3373 11.735C19.271 12.8558 18.7009 13.69 17.6538 14.2242C16.9512 14.5761 16.2221 14.615 15.493 14.3022C14.5386 13.8855 14.0349 12.8558 14.2734 11.8394C14.5649 10.6144 15.3605 9.84524 16.5933 9.57163C17.8526 9.28496 19.0588 10.0147 19.2976 11.305C19.3164 11.3972 19.3217 11.4897 19.328 11.5958Z"
      fill={active ? "url(#paint0_linear_574_1420)" : "white"}
    />
  </svg>
);

const typescriptCodeBlock = `import { verifyKey } from '@unkey/api';

const { result, error } = await verifyKey({
  apiId: "api_123",
  key: "xyz_123"
})

if ( error ) {
  // handle network error
}

if ( !result.valid ) {
  // reject unauthorized request
}

// handle request`;

const nextJsCodeBlock = `import { withUnkey } from '@unkey/nextjs';
export const POST = withUnkey(async (req) => {
  // Process the request here
  // You have access to the typed verification response using \`req.unkey\`
  console.log(req.unkey);
  return new Response('Your API key is valid!');
});`;

const nuxtCodeBlock = `export default defineEventHandler(async (event) => {
  if (!event.context.unkey.valid) {
    throw createError({ statusCode: 403, message: "Invalid API key" })
  }

  // return authorised information
  return {
    // ...
  };
});`;

const pythonCodeBlock = `import asyncio
import os
import unkey

async def main() -> None:
  client = unkey.Client(api_key=os.environ["API_KEY"])
  await client.start()

  result = await client.keys.verify_key("prefix_abc123")

 if result.is_ok:
   print(data.valid)
 else:
   print(result.unwrap_err())`;

const pythonFastAPICodeBlock = `import os
from typing import Any, Dict, Optional

import fastapi  # pip install fastapi
import unkey  # pip install unkey.py
import uvicorn  # pip install uvicorn

app = fastapi.FastAPI()


def key_extractor(*args: Any, **kwargs: Any) -> Optional[str]:
    if isinstance(auth := kwargs.get("authorization"), str):
        return auth.split(" ")[-1]

    return None


@app.get("/protected")
@unkey.protected(os.environ["UNKEY_API_ID"], key_extractor)
async def protected_route(
    *,
    authorization: str = fastapi.Header(None),
    unkey_verification: Any = None,
) -> Dict[str, Optional[str]]:
    assert isinstance(unkey_verification, unkey.ApiKeyVerification)
    assert unkey_verification.valid
    print(unkey_verification.owner_id)
    return {"message": "protected!"}


if __name__ == "__main__":
    uvicorn.run(app)
`;

const honoCodeBlock = `import { Hono } from "hono"
import { UnkeyContext, unkey } from "@unkey/hono";

const app = new Hono<{ Variables: { unkey: UnkeyContext } }>();
app.use("*", unkey());

app.get("/somewhere", (c) => {
  // access the unkey response here to get metadata of the key etc
  const unkey = c.get("unkey")
 return c.text("yo")
})`;

const tsRatelimitCodeBlock = `import { Ratelimit } from "@unkey/ratelimit"

const unkey = new Ratelimit({
  rootKey: process.env.UNKEY_ROOT_KEY,
  namespace: "my-app",
  limit: 10,
  duration: "30s",
  async: true
})

// elsewhere
async function handler(request) {
  const identifier = request.getUserId() // or ip or anything else you want
  
  const ratelimit = await unkey.limit(identifier)
  if (!ratelimit.success){
    return new Response("try again later", { status: 429 })
  }
  
  // handle the request here
  
}`;

const goVerifyKeyCodeBlock = `package main
import (
	"fmt"
	unkey "github.com/WilfredAlmeida/unkey-go/features"
)
func main() {
	apiKey := "key_3ZZ7faUrkfv1YAhffAcnKW74"
	response, err := unkey.KeyVerify(apiKey)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if response.Valid {
		fmt.Println("Key is valid")
	} else {
		fmt.Println("Key is invalid")
	}
}`;
const goCreateKeyCodeBlock = `package main

import (
	"fmt"

	unkey "github.com/WilfredAlmeida/unkey-go/features"
)

func main() {
	// Prepare the request body
	request := unkey.KeyCreateRequest{
		APIId:      "your-api-id",
		Prefix:     "your-prefix",
		ByteLength: 16,
		OwnerId:    "your-owner-id",
		Meta:       map[string]string{"key": "value"},
		Expires:    0,
		Remaining:  0,
		RateLimit: unkey.KeyCreateRateLimit{
			Type:           "fast",
			Limit:          100,
			RefillRate:     10,
			RefillInterval: 60,
		},
	}

	// Provide the authentication token
	authToken := "your-auth-token"

	// Call the KeyCreate function
	response, err := unkey.KeyCreate(request, authToken)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Process the response
	fmt.Println("Key:", response.Key)
	fmt.Println("Key ID:", response.KeyId)
}

`;

const curlVerifyCodeBlock = `curl --request POST \\
  --url https://api.unkey.dev/v1/keys.verifyKey \\
  --header 'Content-Type: application/json' \\
  --data '{
    "apiId": "api_1234",
    "key": "sk_1234",
  }'`;

const curlCreateKeyCodeBlock = `curl --request POST \\
  --url https://api.unkey.dev/v1/keys.createKey \\
  --header 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
  --header 'Content-Type: application/json' \\
  --data '{
    "apiId": "api_123",
    "ownerId": "user_123",
    "expires": ${Date.now() + 7 * 24 * 60 * 60 * 1000},
    "ratelimit": {
      "type": "fast",
      "limit": 10,
      "duration": 60_000
    },
  }'`;

const curlRatelimitCodeBlock = `curl --request POST \
  --url https://api.unkey.dev/v1/ratelimits.limit \
  --header 'Authorization: Bearer <token>' \
  --header 'Content-Type: application/json' \
  --data '{
    "namespace": "email.outbound",
    "identifier": "user_123",
    "limit": 10,
    "duration": 60000,
    "async": true
}'`;

const elixirCodeBlock = `UnkeyElixirSdk.verify_key("xyz_AS5HDkXXPot2MMoPHD8jnL")
# returns
%{"valid" => true,
  "ownerId" => "chronark",
  "meta" => %{
    "hello" => "world"
  }}`;

const rustCodeBlock = `use unkey::models::{VerifyKeyRequest, Wrapped};
use unkey::Client;

async fn verify_key() {
    let api_key = env::var("UNKEY_API_KEY").expect("Environment variable UNKEY_API_KEY not found");
    let c = Client::new(&api_key);
    let req = VerifyKeyRequest::new("test_req", "api_458vdYdbwut5LWABzXZP3Z8jPVas");

    match c.verify_key(req).await {
        Wrapped::Ok(res) => println!("{res:?}"),
        Wrapped::Err(err) => eprintln!("{err:?}"),
    }
}`;

const javaVerifyKeyCodeBlock = `package com.example.myapp;
import com.unkey.unkeysdk.dto.KeyVerifyRequest;
import com.unkey.unkeysdk.dto.KeyVerifyResponse;

@RestController
public class APIController {

    private static IKeyService keyService = new KeyService();

    @PostMapping("/verify")
    public KeyVerifyResponse verifyKey(
        @RequestBody KeyVerifyRequest keyVerifyRequest) {
        // Delegate the creation of the key to the KeyService from the SDK
        return keyService.verifyKey(keyVerifyRequest);
    }
}`;
const javaCreateKeyCodeBlock = `package com.example.myapp;

import com.unkey.unkeysdk.dto.KeyCreateResponse;
import com.unkey.unkeysdk.dto.KeyCreateRequest;

@RestController
public class APIController {

    private static IKeyService keyService = new KeyService();

    @PostMapping("/createKey")
    public KeyCreateResponse createKey(
            @RequestBody KeyCreateRequest keyCreateRequest,
            @RequestHeader("Authorization") String authToken) {
        // Delegate the creation of the key to the KeyService from the SDK
        return keyService.createKey(keyCreateRequest, authToken);
    }
}

`;
type Framework = {
  name: string;
  Icon: React.FC<IconProps>;
  codeBlock: string;
  editorLanguage: string;
};
const languagesList = {
  Typescript: [
    {
      name: "Typescript",
      Icon: TSIcon,
      codeBlock: typescriptCodeBlock,
      editorLanguage: "ts",
    },
    {
      name: "Next.js",
      Icon: TSIcon,
      codeBlock: nextJsCodeBlock,
      editorLanguage: "ts",
    },
    {
      name: "Nuxt",
      codeBlock: nuxtCodeBlock,
      Icon: TSIcon,
      editorLanguage: "ts",
    },
    {
      name: "Hono",
      Icon: TSIcon,
      codeBlock: honoCodeBlock,
      editorLanguage: "ts",
    },
    {
      name: "Ratelimiting",
      Icon: TSIcon,
      codeBlock: tsRatelimitCodeBlock,
      editorLanguage: "ts",
    },
  ],
  Python: [
    {
      name: "Python",
      Icon: PythonIcon,
      codeBlock: pythonCodeBlock,
      editorLanguage: "python",
    },
    {
      name: "FastAPI",
      Icon: PythonIcon,
      codeBlock: pythonFastAPICodeBlock,
      editorLanguage: "python",
    },
  ],
  Golang: [
    {
      name: "Verify key",
      Icon: GoIcon,
      codeBlock: goVerifyKeyCodeBlock,
      editorLanguage: "go",
    },
    {
      name: "Create key",
      Icon: GoIcon,
      codeBlock: goCreateKeyCodeBlock,
      editorLanguage: "go",
    },
  ],
  Java: [
    {
      name: "Verify key",
      Icon: JavaIcon,
      codeBlock: javaVerifyKeyCodeBlock,
      editorLanguage: "ts",
    },
    {
      name: "Create key",
      Icon: JavaIcon,
      codeBlock: javaCreateKeyCodeBlock,
      editorLanguage: "ts",
    },
  ],
  Elixir: [
    {
      name: "Verify key",
      Icon: ElixirIcon,
      codeBlock: elixirCodeBlock,
      editorLanguage: "ts",
    },
  ],
  Rust: [
    {
      name: "Verify key",
      Icon: RustIcon,
      codeBlock: rustCodeBlock,
      editorLanguage: "rust",
    },
  ],
  Curl: [
    {
      name: "Verify key",
      Icon: CurlIcon,
      codeBlock: curlVerifyCodeBlock,
      editorLanguage: "tsx",
    },
    {
      name: "Create key",
      Icon: CurlIcon,
      codeBlock: curlCreateKeyCodeBlock,
      editorLanguage: "tsx",
    },
    {
      name: "Ratelimit",
      Icon: CurlIcon,
      codeBlock: curlRatelimitCodeBlock,
      editorLanguage: "tsx",
    },
  ],
} as const satisfies {
  [key: string]: Framework[];
};

// const TabsContent = React.forwardRef<
//   React.ElementRef<typeof TabsPrimitive.Content>,
//   React.ComponentPropsWithoutRef<typeof TabsPrimitive.Content>
// >(({ className, ...props }, ref) => (
//   <TabsPrimitive.Content
//     ref={ref}
//     className={cn(
//       "mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
//       className,
//     )}
//     {...props}
//   />
// ));
// TabsContent.displayName = TabsPrimitive.Content.displayName;

type Props = {
  className?: string;
};

type Language = "Typescript" | "Python" | "Rust" | "Golang" | "Curl" | "Elixir" | "Java";

// TODO extract this automatically from our languages array
type FrameworkName = (typeof languagesList)[Language][number]["name"];

export const CodeExamples: React.FC<Props> = ({ className }) => {
  const [language, setLanguage] = useState<Language>("Typescript");
  const [framework, setFramework] = useState<FrameworkName>("Typescript");
  const [languageHover, setLanguageHover] = useState("Typescript");
  function getLanguage({ language, framework }: { language: Language; framework: FrameworkName }) {
    const frameworks = languagesList[language];
    const currentFramework = frameworks.find((f) => f.name === framework);
    return currentFramework?.editorLanguage || "tsx";
  }

  useEffect(() => {
    setFramework(languagesList[language].at(0)!.name);
  }, [language]);

  function getCodeBlock({ language, framework }: { language: Language; framework: FrameworkName }) {
    const frameworks = languagesList[language];
    const currentFramework = frameworks.find((f) => f.name === framework);
    return currentFramework?.codeBlock || "";
  }

  const LanguageTrigger = React.forwardRef<
    React.ElementRef<typeof TabsPrimitive.Trigger>,
    React.ComponentPropsWithoutRef<typeof TabsPrimitive.Trigger>
  >(({ className, value, ...props }, ref) => (
    <TabsPrimitive.Trigger
      ref={ref}
      value={value}
      onMouseEnter={() => setLanguageHover(value)}
      className={cn(
        "inline-flex items-center gap-1 justify-center whitespace-nowrap rounded-t-lg px-3  py-1.5 text-sm transition-all hover:text-white/80 disabled:pointer-events-none disabled:opacity-50 bg-gradient-to-t from-black to-black data-[state=active]:from-white/10 border border-b-0 text-white/30 data-[state=active]:text-white border-[#454545] font-light",
        className,
      )}
      {...props}
    />
  ));
  LanguageTrigger.displayName = TabsPrimitive.Trigger.displayName;
  const [copied, setCopied] = useState(false);
  return (
    <section className={className}>
      <SectionTitle
        label="Code"
        title="Any language, any framework, always secure"
        text="Add authentication to your APIs in a few lines of code. We provide SDKs for a range of languages and frameworks, and an intuitive REST API with public OpenAPI spec."
        align="center"
        className="relative"
      >
        <div className="absolute bottom-32 left-[-50px]">
          <MeteorLines className="ml-2 fade-in-0" delay={3} number={1} />
          <MeteorLines className="ml-10 fade-in-40" delay={0} number={1} />
          <MeteorLines className="ml-16 fade-in-100" delay={5} number={1} />
        </div>
        <div className="absolute bottom-32 right-[200px]">
          <MeteorLines className="ml-2 fade-in-0" delay={4} number={1} />
          <MeteorLines className="ml-10 fade-in-40" delay={0} number={1} />
          <MeteorLines className="ml-16 fade-in-100" delay={2} number={1} />
        </div>
        <div className="mt-10">
          <div className="flex gap-6 pb-14">
            <Link key="get-started" href="https://app.unkey.com">
              <PrimaryButton shiny label="Get Started" IconRight={ChevronRight} />
            </Link>
            <Link key="docs" href="/docs">
              <SecondaryButton label="Visit the docs" IconRight={ChevronRight} />
            </Link>
          </div>
        </div>
      </SectionTitle>
      <div className="relative w-full rounded-4xl border-[.75px] border-white/10 bg-gradient-to-b from-[#111111] to-black border-t-[.75px] border-t-white/20">
        <div aria-hidden className="absolute inset-x-0 -top-[432px] bottom-[calc(100%-2rem)]">
          <HighlightAbove className="w-full h-full" />
        </div>
        <Tabs
          defaultValue={language}
          onValueChange={(l) => setLanguage(l as Language)}
          className="relative flex items-end h-16 px-4 border rounded-tr-3xl rounded-tl-3xl border-white/10 editor-top-gradient"
        >
          <TabsPrimitive.List className="flex items-end gap-4 overflow-x-auto scrollbar-hidden">
            <LanguageTrigger value="Typescript">
              <TSIcon active={languageHover === "Typescript" || language === "Typescript"} />
              Typescript
            </LanguageTrigger>
            <LanguageTrigger value="Python">
              <PythonIcon active={languageHover === "Python" || language === "Python"} />
              Python
            </LanguageTrigger>
            <LanguageTrigger value="Golang">
              <GoIcon active={languageHover === "Golang" || language === "Golang"} />
              Golang
            </LanguageTrigger>
            <LanguageTrigger value="Curl">
              <CurlIcon active={languageHover === "Curl" || language === "Curl"} />
              Curl
            </LanguageTrigger>
            <LanguageTrigger value="Elixir">
              <ElixirIcon active={languageHover === "Elixir" || language === "Elixir"} />
              Elixir
            </LanguageTrigger>
            <LanguageTrigger value="Rust">
              <RustIcon active={languageHover === "Rust" || language === "Rust"} />
              Rust
            </LanguageTrigger>
            <LanguageTrigger value="Java">
              <JavaIcon active={languageHover === "Java" || language === "Java"} />
              Java
            </LanguageTrigger>
          </TabsPrimitive.List>
        </Tabs>
        <div className="flex flex-col sm:flex-row overflow-x-auto scrollbar-hidden sm:h-[520px]">
          <FrameworkSwitcher
            frameworks={languagesList[language]}
            currentFramework={framework}
            setFramework={setFramework}
          />
          <div className="relative flex w-full pt-4 pb-8 pl-8 font-mono text-xs text-white sm:text-sm">
            <Editor
              language={getLanguage({ language, framework })}
              theme={editorTheme}
              codeBlock={getCodeBlock({ language, framework })}
            />
            <button
              type="button"
              aria-label="Copy code"
              className="absolute hidden cursor-pointer top-5 right-5 lg:flex"
              onClick={() => {
                navigator.clipboard.writeText(getCodeBlock({ language, framework }));
                setCopied(true);
                setTimeout(() => {
                  setCopied(false);
                }, 2000);
              }}
            >
              {copied ? (
                <svg className="checkmark" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 52 52">
                  <circle className="checkmark__circle" cx="26" cy="26" r="25" fill="none" />
                  <path className="checkmark__check" fill="none" d="M14.1 27.2l7.1 7.2 16.7-16.8" />
                </svg>
              ) : (
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  width="24"
                  height="24"
                  viewBox="0 0 24 24"
                  fill="none"
                  className=""
                >
                  <path
                    opacity="0.3"
                    fillRule="evenodd"
                    clipRule="evenodd"
                    d="M13 5.00002C13.4886 5.00002 13.6599 5.00244 13.7927 5.02884C14.3877 5.1472 14.8528 5.61235 14.9712 6.20738C14.9976 6.34011 15 6.5114 15 7.00002H16L16 6.94215V6.94213C16.0001 6.53333 16.0002 6.25469 15.952 6.01229C15.7547 5.02057 14.9795 4.24532 13.9877 4.04806C13.7453 3.99984 13.4667 3.99991 13.0579 4.00001L13 4.00002H7.70002H7.67861C7.13672 4.00001 6.69965 4.00001 6.34571 4.02893C5.98128 4.0587 5.66119 4.12161 5.36504 4.2725C4.89464 4.51219 4.51219 4.89464 4.2725 5.36504C4.12161 5.66119 4.0587 5.98128 4.02893 6.34571C4.00001 6.69965 4.00001 7.13672 4.00002 7.67862V7.70002V13L4.00001 13.0579C3.99991 13.4667 3.99984 13.7453 4.04806 13.9877C4.24532 14.9795 5.02057 15.7547 6.01229 15.952C6.25469 16.0002 6.53333 16.0001 6.94213 16H6.94215L7.00002 16V15C6.5114 15 6.34011 14.9976 6.20738 14.9712C5.61235 14.8528 5.1472 14.3877 5.02884 13.7927C5.00244 13.6599 5.00002 13.4886 5.00002 13V7.70002C5.00002 7.13172 5.00041 6.73556 5.02561 6.42714C5.05033 6.12455 5.09642 5.95071 5.16351 5.81903C5.30732 5.53679 5.53679 5.30732 5.81903 5.16351C5.95071 5.09642 6.12455 5.05033 6.42714 5.02561C6.73556 5.00041 7.13172 5.00002 7.70002 5.00002H13ZM11.7 8.00002H11.6786C11.1367 8.00001 10.6996 8.00001 10.3457 8.02893C9.98128 8.0587 9.66119 8.12161 9.36504 8.2725C8.89464 8.51219 8.51219 8.89464 8.2725 9.36504C8.12161 9.66119 8.0587 9.98128 8.02893 10.3457C8.00001 10.6996 8.00001 11.1367 8.00002 11.6786V11.7V16.3V16.3214C8.00001 16.8633 8.00001 17.3004 8.02893 17.6543C8.0587 18.0188 8.12161 18.3388 8.2725 18.635C8.51219 19.1054 8.89464 19.4879 9.36504 19.7275C9.66119 19.8784 9.98128 19.9413 10.3457 19.9711C10.6996 20 11.1366 20 11.6785 20H11.6786H11.7H16.3H16.3214H16.3216C16.8634 20 17.3004 20 17.6543 19.9711C18.0188 19.9413 18.3388 19.8784 18.635 19.7275C19.1054 19.4879 19.4879 19.1054 19.7275 18.635C19.8784 18.3388 19.9413 18.0188 19.9711 17.6543C20 17.3004 20 16.8634 20 16.3216V16.3214V16.3V11.7V11.6786V11.6785C20 11.1366 20 10.6996 19.9711 10.3457C19.9413 9.98128 19.8784 9.66119 19.7275 9.36504C19.4879 8.89464 19.1054 8.51219 18.635 8.2725C18.3388 8.12161 18.0188 8.0587 17.6543 8.02893C17.3004 8.00001 16.8633 8.00001 16.3214 8.00002H16.3H11.7ZM9.81903 9.16351C9.95071 9.09642 10.1246 9.05033 10.4271 9.02561C10.7356 9.00041 11.1317 9.00002 11.7 9.00002H16.3C16.8683 9.00002 17.2645 9.00041 17.5729 9.02561C17.8755 9.05033 18.0493 9.09642 18.181 9.16351C18.4632 9.30732 18.6927 9.53679 18.8365 9.81903C18.9036 9.95071 18.9497 10.1246 18.9744 10.4271C18.9996 10.7356 19 11.1317 19 11.7V16.3C19 16.8683 18.9996 17.2645 18.9744 17.5729C18.9497 17.8755 18.9036 18.0493 18.8365 18.181C18.6927 18.4632 18.4632 18.6927 18.181 18.8365C18.0493 18.9036 17.8755 18.9497 17.5729 18.9744C17.2645 18.9996 16.8683 19 16.3 19H11.7C11.1317 19 10.7356 18.9996 10.4271 18.9744C10.1246 18.9497 9.95071 18.9036 9.81903 18.8365C9.53679 18.6927 9.30732 18.4632 9.16351 18.181C9.09642 18.0493 9.05033 17.8755 9.02561 17.5729C9.00041 17.2645 9.00002 16.8683 9.00002 16.3V11.7C9.00002 11.1317 9.00041 10.7356 9.02561 10.4271C9.05033 10.1246 9.09642 9.95071 9.16351 9.81903C9.30732 9.53679 9.53679 9.30732 9.81903 9.16351Z"
                    fill="url(#paint0_linear_840_3800)"
                  />
                  <defs>
                    <linearGradient
                      id="paint0_linear_840_3800"
                      x1="4.15606"
                      y1="2.27462"
                      x2="4.15606"
                      y2="20.9494"
                      gradientUnits="userSpaceOnUse"
                    >
                      <stop stopColor="white" stopOpacity="0.4" />
                      <stop offset="1" stopColor="white" />
                    </linearGradient>
                  </defs>
                </svg>
              )}
            </button>
          </div>
        </div>
      </div>
    </section>
  );
};

function FrameworkSwitcher({
  frameworks,
  currentFramework,
  setFramework,
}: {
  frameworks: Framework[];
  currentFramework: FrameworkName;
  setFramework: React.Dispatch<React.SetStateAction<FrameworkName>>;
}) {
  return (
    <div className="flex flex-col justify-between sm:w-[216px] text-white text-sm pt-6 px-4 font-mono md:border-r md:border-white/10">
      <div className="flex items-center space-x-2 sm:flex-col sm:space-x-0 sm:space-y-2">
        {frameworks.map((framework) => (
          <button
            key={framework.name}
            type="button"
            onClick={() => {
              setFramework(framework.name as FrameworkName);
            }}
            className={cn(
              "flex items-center cursor-pointer hover:bg-white/10 py-1 px-2 rounded-lg w-[184px] ",
              {
                "bg-white/10 text-white": currentFramework === framework.name,
                "text-white/40": currentFramework !== framework.name,
              },
            )}
          >
            <div>{framework.name}</div>
          </button>
        ))}
      </div>
    </div>
  );
}

const HighlightAbove: React.FC<{ className: string }> = ({ className }) => (
  <svg
    width="1086"
    height="620"
    viewBox="0 0 1086 620"
    fill="none"
    xmlns="http://www.w3.org/2000/svg"
    className={className}
  >
    <mask id="path-1-inside-1_830_9506" fill="white">
      <path d="M31 210H1055V642H31V210Z" />
    </mask>
    <path d="M31 210H1055V642H31V210Z" fill="url(#paint0_radial_830_9506)" />
    <defs>
      <radialGradient
        id="paint0_radial_830_9506"
        cx="0"
        cy="0"
        r="1"
        gradientUnits="userSpaceOnUse"
        gradientTransform="translate(543 642) rotate(-90) scale(409.05 969.6)"
      >
        <stop stopColor="white" stopOpacity="0.2" />
        <stop offset="0.554455" stopColor="white" stopOpacity="0" />
      </radialGradient>
      <linearGradient
        id="paint1_linear_830_9506"
        x1="543"
        y1="210"
        x2="543"
        y2="642"
        gradientUnits="userSpaceOnUse"
      >
        <stop stopColor="white" stopOpacity="0" />
        <stop offset="1" stopColor="white" stopOpacity="0.3" />
      </linearGradient>
      <linearGradient
        id="paint2_linear_830_9506"
        x1="543"
        y1="0"
        x2="543"
        y2="432"
        gradientUnits="userSpaceOnUse"
      >
        <stop offset="0.765625" stopColor="white" stopOpacity="0" />
        <stop offset="1" stopColor="white" stopOpacity="0.4" />
      </linearGradient>
      <linearGradient
        id="paint3_linear_830_9506"
        x1="543"
        y1="100"
        x2="543"
        y2="532"
        gradientUnits="userSpaceOnUse"
      >
        <stop offset="0.697917" stopColor="white" stopOpacity="0" />
        <stop offset="1" stopColor="white" stopOpacity="0.3" />
      </linearGradient>
    </defs>
  </svg>
);
