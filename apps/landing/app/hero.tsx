"use client";
import { YoutubeEmbed } from "@/components/youtube-embed";
import { motion } from "framer-motion";
import Image from "next/image";
import { HeroMainSection } from "./hero/hero-main-section";

import mainboard from "../images/mainboard.svg";
export const Hero: React.FC = () => {
  const containerVariants = {
    hidden: {},
    visible: {
      transition: {
        staggerChildren: 0.3,
      },
    },
  };

  const childVariants = {
    hidden: { opacity: 0, y: 25 },
    visible: {
      opacity: 1,
      y: 0,
      transition: { duration: 0.6, ease: "easeOut" },
    },
  };

  return (
    <motion.div
      className="relative flex flex-col items-center justify-between mt-48 xl:flex-row xl:items-start"
      variants={containerVariants}
      initial="hidden"
      animate="visible"
    >
      <motion.div variants={childVariants}>
        <HeroMainSection />
      </motion.div>
      <div className="relative ">
        <Image
          src={mainboard}
          alt="Animated SVG showing computer circuits lighting up"
          className="absolute hidden xl:right-32 xl:flex -z-10 xl:-top-56"
          style={{ transform: "scale(2)" }}
          priority
        />

        <motion.div variants={childVariants}>
          <YoutubeEmbed />
        </motion.div>
      </div>
      <SubHeroMainboardStuff className="absolute hidden md:flex left-[400px] top-[250px]" />
    </motion.div>
  );
};

function SubHeroMainboardStuff({ className }: { className?: string }) {
  return (
    <svg
      width="908"
      height="357"
      viewBox="0 0 908 357"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={className}
    >
      <mask
        id="mask0_1_215"
        style={{ maskType: "alpha" }}
        maskUnits="userSpaceOnUse"
        x="0"
        y="0"
        width="1072"
        height="324"
      >
        <rect
          width="1072"
          height="323"
          transform="translate(0 0.25)"
          fill="url(#paint0_radial_1_215)"
        />
      </mask>
      <g mask="url(#mask0_1_215)">
        <path
          d="M1072 194.25H735.604C731.69 194.25 729.734 194.25 727.892 193.808C726.26 193.416 724.699 192.77 723.267 191.892C721.653 190.903 720.269 189.519 717.502 186.752L640 109.25"
          stroke="white"
          strokeOpacity="0.08"
          strokeWidth="0.5"
        />
        <rect x="828" y="149" width="21" height="24" rx="3" fill="white" fillOpacity="0.16" />
        <rect x="831" y="163" width="15" height="6" rx="1.5" fill="white" fillOpacity="0.16" />
        <rect x="746" y="151" width="8" height="2" fill="white" fillOpacity="0.18" />
        <rect x="746" y="151" width="8" height="1" fill="white" fillOpacity="0.14" />
        <rect x="746" y="161" width="8" height="2" fill="white" fillOpacity="0.18" />
        <rect x="746" y="161" width="8" height="1" fill="white" fillOpacity="0.14" />
        <rect x="747" y="155" width="2" height="4" fill="white" fillOpacity="0.18" />
        <rect x="747" y="155" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="751" y="155" width="2" height="4" fill="white" fillOpacity="0.18" />
        <rect x="751" y="155" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="746" y="171" width="8" height="2" fill="white" fillOpacity="0.18" />
        <rect x="746" y="171" width="8" height="1" fill="white" fillOpacity="0.14" />
        <rect x="746" y="181" width="8" height="2" fill="white" fillOpacity="0.18" />
        <rect x="746" y="181" width="8" height="1" fill="white" fillOpacity="0.14" />
        <rect x="747" y="175" width="2" height="4" fill="white" fillOpacity="0.18" />
        <rect x="747" y="175" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="751" y="175" width="2" height="4" fill="white" fillOpacity="0.18" />
        <rect x="751" y="175" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="762" y="171" width="8" height="2" fill="white" fillOpacity="0.18" />
        <rect x="762" y="171" width="8" height="1" fill="white" fillOpacity="0.14" />
        <rect x="762" y="181" width="8" height="2" fill="white" fillOpacity="0.18" />
        <rect x="762" y="181" width="8" height="1" fill="white" fillOpacity="0.14" />
        <rect x="763" y="175" width="2" height="4" fill="white" fillOpacity="0.18" />
        <rect x="763" y="175" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="767" y="175" width="2" height="4" fill="white" fillOpacity="0.18" />
        <rect x="767" y="175" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="778" y="171" width="8" height="2" fill="white" fillOpacity="0.18" />
        <rect x="778" y="171" width="8" height="1" fill="white" fillOpacity="0.14" />
        <rect x="778" y="181" width="8" height="2" fill="white" fillOpacity="0.18" />
        <rect x="778" y="181" width="8" height="1" fill="white" fillOpacity="0.14" />
        <rect x="779" y="175" width="2" height="4" fill="white" fillOpacity="0.18" />
        <rect x="779" y="175" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="783" y="175" width="2" height="4" fill="white" fillOpacity="0.18" />
        <rect x="783" y="175" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="762" y="151" width="8" height="2" fill="white" fillOpacity="0.18" />
        <rect x="762" y="151" width="8" height="1" fill="white" fillOpacity="0.14" />
        <rect x="762" y="161" width="8" height="2" fill="white" fillOpacity="0.18" />
        <rect x="762" y="161" width="8" height="1" fill="white" fillOpacity="0.14" />
        <rect x="763" y="155" width="2" height="4" fill="white" fillOpacity="0.18" />
        <rect x="763" y="155" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="767" y="155" width="2" height="4" fill="white" fillOpacity="0.18" />
        <rect x="767" y="155" width="2" height="1" fill="white" fillOpacity="0.14" />
        <path d="M869 194.5V323" stroke="white" strokeOpacity="0.04" strokeWidth="2" />
        <path d="M918 0.25H807.25V194" stroke="white" strokeOpacity="0.08" strokeWidth="0.5" />
        <path
          d="M924.25 323.25H807.25V194.5"
          stroke="white"
          strokeOpacity="0.08"
          strokeWidth="0.5"
        />
        <path
          d="M0 237.25H312.084C314.115 237.25 315.131 237.25 316.105 237.055C316.968 236.881 317.806 236.595 318.594 236.202C319.483 235.76 320.286 235.137 321.891 233.892L351.296 211.081C353.114 209.67 354.023 208.965 354.678 208.081C355.258 207.298 355.69 206.416 355.953 205.478C356.25 204.419 356.25 203.268 356.25 200.967V105"
          stroke="white"
          strokeOpacity="0.08"
          strokeWidth="0.5"
        />
        <path d="M356 157.25H177.25V237" stroke="white" strokeOpacity="0.08" strokeWidth="0.5" />
        <rect x="321" y="142" width="2" height="6" fill="white" fillOpacity="0.18" />
        <rect x="326" y="142" width="2" height="6" fill="white" fillOpacity="0.18" />
        <rect x="331" y="142" width="2" height="6" fill="white" fillOpacity="0.18" />
        <rect x="336" y="142" width="2" height="6" fill="white" fillOpacity="0.18" />
        <rect x="341" y="142" width="2" height="6" fill="white" fillOpacity="0.18" />
        <rect x="346" y="142" width="2" height="6" fill="white" fillOpacity="0.18" />
        <path d="M177.25 7.5V157" stroke="white" strokeOpacity="0.08" strokeWidth="0.5" />
        <rect opacity="0.02" x="183" y="163" width="75" height="68" rx="0.5" fill="white" />
        <rect opacity="0.05" x="187" y="189" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="187" y="182" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="187" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="187" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="187" y="167" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="187" y="196" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="187" y="203" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="187" y="210" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="187" y="217" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="187" y="224" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="195" y="189" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="195" y="182" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="195" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="195" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="195" y="167" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="195" y="196" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.1" x="195" y="203" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="195" y="210" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="195" y="217" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="195" y="224" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="203" y="189" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="203" y="182" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="203" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="203" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="203" y="167" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="203" y="196" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="203" y="203" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="203" y="210" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="203" y="217" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.15" x="203" y="224" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.1" x="211" y="189" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="211" y="182" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="211" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="211" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="211" y="167" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="211" y="196" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="211" y="203" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="211" y="210" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="211" y="217" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="211" y="224" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="219" y="189" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="219" y="182" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="219" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="219" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="219" y="167" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="219" y="196" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="219" y="203" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.15" x="219" y="210" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="219" y="217" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="219" y="224" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="227" y="189" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="227" y="182" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="227" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="227" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="227" y="167" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="227" y="196" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="227" y="203" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="227" y="210" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="227" y="217" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="227" y="224" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="235" y="189" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="235" y="182" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="235" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="235" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="235" y="167" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="235" y="196" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.15" x="235" y="203" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="235" y="210" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="235" y="217" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="235" y="224" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.1" x="243" y="189" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="243" y="182" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="243" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="243" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="243" y="167" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="243" y="196" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="243" y="203" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="243" y="210" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="243" y="217" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.1" x="243" y="224" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="251" y="189" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="251" y="182" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="251" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="251" y="174" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="251" y="167" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="251" y="196" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="251" y="203" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="251" y="210" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="251" y="217" width="3" height="3" rx="0.5" fill="white" />
        <rect opacity="0.05" x="251" y="224" width="3" height="3" rx="0.5" fill="white" />
        <rect x="107" y="215" width="2" height="8" fill="white" fillOpacity="0.18" />
        <rect x="111" y="215" width="2" height="8" fill="white" fillOpacity="0.18" />
        <rect x="102" y="218" width="2" height="2" fill="white" fillOpacity="0.18" />
        <rect x="102" y="218" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="107" y="215" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="111" y="215" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="107" y="199" width="2" height="8" fill="white" fillOpacity="0.18" />
        <rect x="111" y="199" width="2" height="8" fill="white" fillOpacity="0.18" />
        <rect x="102" y="202" width="2" height="2" fill="white" fillOpacity="0.18" />
        <rect x="102" y="202" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="107" y="199" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="111" y="199" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="107" y="183" width="2" height="8" fill="white" fillOpacity="0.18" />
        <rect x="111" y="183" width="2" height="8" fill="white" fillOpacity="0.18" />
        <rect x="102" y="186" width="2" height="2" fill="white" fillOpacity="0.18" />
        <rect x="102" y="186" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="107" y="183" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="111" y="183" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="126" y="215" width="2" height="8" fill="white" fillOpacity="0.18" />
        <rect x="130" y="215" width="2" height="8" fill="white" fillOpacity="0.18" />
        <rect x="121" y="218" width="2" height="2" fill="white" fillOpacity="0.18" />
        <rect x="121" y="218" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="126" y="215" width="2" height="1" fill="white" fillOpacity="0.14" />
        <rect x="130" y="215" width="2" height="1" fill="white" fillOpacity="0.14" />
        <path
          d="M286 7.25H158.67C154.111 7.25 151.831 7.25 149.729 7.83406C147.867 8.35146 146.114 9.20191 144.555 10.3442C142.796 11.6337 141.385 13.4244 138.563 17.0056L92.7424 75.1548C90.7062 77.7388 89.6881 79.0309 88.9644 80.4657C88.3222 81.7388 87.8532 83.092 87.5696 84.4893C87.25 86.0643 87.25 87.7092 87.25 90.9992V237"
          stroke="white"
          strokeOpacity="0.08"
          strokeWidth="0.5"
        />
        <g clipPath="url(#clip0_1_215)">
          <path
            d="M744 288C749.333 282.667 852.889 179.111 904 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
          <path
            d="M752 288C757.333 282.667 860.889 179.111 912 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
          <path
            d="M760 288C765.333 282.667 868.889 179.111 920 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
          <path
            d="M768 288C773.333 282.667 876.889 179.111 928 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
          <path
            d="M776 288C781.333 282.667 884.889 179.111 936 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
          <path
            d="M784 288C789.333 282.667 892.889 179.111 944 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
          <path
            d="M792 288C797.333 282.667 900.889 179.111 952 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
          <path
            d="M800 288C805.333 282.667 908.889 179.111 960 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
          <path
            d="M808 288C813.333 282.667 916.889 179.111 968 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
          <path
            d="M816 288C821.333 282.667 924.889 179.111 976 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
          <path
            d="M824 288C829.333 282.667 932.889 179.111 984 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
          <path
            d="M832 288C837.333 282.667 940.889 179.111 992 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
          <path
            d="M840 288C845.333 282.667 948.889 179.111 1000 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
          <path
            d="M848 288C853.333 282.667 956.889 179.111 1008 128"
            stroke="white"
            strokeOpacity="0.3"
            strokeWidth="0.3"
          />
        </g>
        <rect x="818" y="298" width="2" height="6" fill="white" fillOpacity="0.18" />
        <rect x="823" y="298" width="2" height="6" fill="white" fillOpacity="0.18" />
        <rect x="828" y="298" width="2" height="6" fill="white" fillOpacity="0.18" />
        <rect x="833" y="298" width="2" height="6" fill="white" fillOpacity="0.18" />
        <rect x="838" y="298" width="2" height="6" fill="white" fillOpacity="0.18" />
        <rect x="843" y="298" width="2" height="6" fill="white" fillOpacity="0.18" />
      </g>
      <path d="M566.25 357V109" stroke="url(#paint1_linear_1_215)" strokeWidth="0.5" />
      <path d="M546.25 357V109" stroke="url(#paint2_linear_1_215)" strokeWidth="0.5" />
      <path d="M526.25 357V109" stroke="url(#paint3_linear_1_215)" strokeWidth="0.5" />
      <path
        d="M526.25 357V109"
        stroke="url(#paint4_angular_1_215)"
        strokeOpacity="0.5"
        strokeWidth="0.5"
      />
      <path d="M506.25 357V109" stroke="url(#paint5_linear_1_215)" strokeWidth="0.5" />
      <path d="M486.25 357V109" stroke="url(#paint6_linear_1_215)" strokeWidth="0.5" />
      <path
        d="M486.25 357V109"
        stroke="url(#paint7_angular_1_215)"
        strokeOpacity="0.5"
        strokeWidth="0.5"
      />
      <path d="M496.25 357V109" stroke="url(#paint8_linear_1_215)" strokeWidth="0.5" />
      <path d="M516.25 357V109" stroke="url(#paint9_linear_1_215)" strokeWidth="0.5" />
      <path d="M536.25 357V109" stroke="url(#paint10_linear_1_215)" strokeWidth="0.5" />
      <path d="M556.25 357V109" stroke="url(#paint11_linear_1_215)" strokeWidth="0.5" />
      <path d="M576.25 357V109" stroke="url(#paint12_linear_1_215)" strokeWidth="0.5" />
      <path d="M585.25 357V109" stroke="url(#paint13_linear_1_215)" strokeWidth="0.5" />
      <path
        d="M585.25 357V109"
        stroke="url(#paint14_angular_1_215)"
        strokeOpacity="0.5"
        strokeWidth="0.5"
      />
      <defs>
        <radialGradient
          id="paint0_radial_1_215"
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(536 161.5) rotate(90) scale(161.5 536)"
        >
          <stop stopColor="white" />
          <stop offset="0.496904" stopColor="white" />
          <stop offset="1" stopColor="white" stopOpacity="0" />
        </radialGradient>
        <linearGradient
          id="paint1_linear_1_215"
          x1="566.75"
          y1="109"
          x2="566.75"
          y2="357"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.15" />
          <stop offset="1" stopColor="white" stopOpacity="0.05" />
        </linearGradient>
        <linearGradient
          id="paint2_linear_1_215"
          x1="546.75"
          y1="109"
          x2="546.75"
          y2="357"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.15" />
          <stop offset="1" stopColor="white" stopOpacity="0.05" />
        </linearGradient>
        <linearGradient
          id="paint3_linear_1_215"
          x1="526.75"
          y1="109"
          x2="526.75"
          y2="357"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.15" />
          <stop offset="1" stopColor="white" stopOpacity="0.05" />
        </linearGradient>
        <radialGradient
          id="paint4_angular_1_215"
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(483 306.421) scale(55 213.125)"
        >
          <stop stopColor="white" />
          <stop offset="0.0001" stopColor="white" stopOpacity="0" />
          <stop offset="0.199397" stopColor="white" stopOpacity="0" />
          <stop offset="0.939101" stopColor="white" stopOpacity="0" />
        </radialGradient>
        <linearGradient
          id="paint5_linear_1_215"
          x1="506.75"
          y1="109"
          x2="506.75"
          y2="357"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.15" />
          <stop offset="1" stopColor="white" stopOpacity="0.05" />
        </linearGradient>
        <linearGradient
          id="paint6_linear_1_215"
          x1="486.75"
          y1="109"
          x2="486.75"
          y2="357"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.15" />
          <stop offset="1" stopColor="white" stopOpacity="0.05" />
        </linearGradient>
        <radialGradient
          id="paint7_angular_1_215"
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(445 257.474) scale(72 117.474)"
        >
          <stop stopColor="white" />
          <stop offset="0.0001" stopColor="white" stopOpacity="0" />
          <stop offset="0.199397" stopColor="white" stopOpacity="0" />
          <stop offset="0.856266" stopColor="white" stopOpacity="0" />
        </radialGradient>
        <linearGradient
          id="paint8_linear_1_215"
          x1="496.75"
          y1="109"
          x2="496.75"
          y2="357"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.15" />
          <stop offset="1" stopColor="white" stopOpacity="0.05" />
        </linearGradient>
        <linearGradient
          id="paint9_linear_1_215"
          x1="516.75"
          y1="109"
          x2="516.75"
          y2="357"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.15" />
          <stop offset="1" stopColor="white" stopOpacity="0.05" />
        </linearGradient>
        <linearGradient
          id="paint10_linear_1_215"
          x1="536.75"
          y1="109"
          x2="536.75"
          y2="357"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.15" />
          <stop offset="1" stopColor="white" stopOpacity="0.05" />
        </linearGradient>
        <linearGradient
          id="paint11_linear_1_215"
          x1="556.75"
          y1="109"
          x2="556.75"
          y2="357"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.15" />
          <stop offset="1" stopColor="white" stopOpacity="0.05" />
        </linearGradient>
        <linearGradient
          id="paint12_linear_1_215"
          x1="576.75"
          y1="109"
          x2="576.75"
          y2="357"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.15" />
          <stop offset="1" stopColor="white" stopOpacity="0.05" />
        </linearGradient>
        <linearGradient
          id="paint13_linear_1_215"
          x1="585.75"
          y1="109"
          x2="585.75"
          y2="357"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="white" stopOpacity="0.15" />
          <stop offset="1" stopColor="white" stopOpacity="0.05" />
        </linearGradient>
        <radialGradient
          id="paint14_angular_1_215"
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(549.5 213.625) scale(45 174.375)"
        >
          <stop stopColor="white" />
          <stop offset="0.0001" stopColor="white" stopOpacity="0" />
          <stop offset="0.199397" stopColor="white" stopOpacity="0" />
          <stop offset="0.946685" stopColor="white" stopOpacity="0" />
        </radialGradient>
        <clipPath id="clip0_1_215">
          <rect width="40" height="77" fill="white" transform="translate(818 205)" />
        </clipPath>
      </defs>
    </svg>
  );
}
